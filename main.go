package main

import (
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jordan-wright/email"
	_ "github.com/mattn/go-sqlite3"
)

var (
	XUuid         = os.Getenv("XUuid")
	Authorization = os.Getenv("Authorization")
	PatId         = os.Getenv("PatId")
	UserId        = os.Getenv("UserId")

	DytApiHost = os.Getenv("DytApiHost")
)

const (
	KeyWord = "九价"

	EmailUser = ""
	EmailPass = ""
	EmailTo1  = ""
	EmailTo2  = ""

	IsAppointment = true
	IsSending     = false

	// AppointCount Number of retries after a failed appointment
	AppointCount = 5
	// AppointSleep Delay in milliseconds after a failed appointment
	AppointSleep = 500

	// ErrorCount The program will panic when the number of errors exceeds this value
	ErrorCount = 5

	ThreadNum = 10

	IsDebug = false
)

// Response json of hospital list
type hosResp struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		HosLogo    string `json:"hos_logo"`
		HosName    string `json:"hos_name"`
		HosAddress string `json:"hos_address"`
		HosId      string `json:"hos_id"`
		Status     int64  `json:"status"`
		Doctor     []struct {
			DepId int64 `json:"dep_id"`
			DocId int64 `json:"doc_id"`
		} `json:"doctor"`
		HosCode  int64       `json:"hos_code"`
		HosCode2 interface{} `json:"hos_code2"`
		Sort     int64       `json:"sort"`
	} `json:"data"`
}

// Response json of hospital detail list
type hosDetailResp struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ClassList       []interface{} `json:"class_list"`
		DepId           string        `json:"dep_id"`
		DepName         string        `json:"dep_name"`
		DocAvatar       string        `json:"doc_avatar"`
		DocGood         string        `json:"doc_good"`
		DocId           string        `json:"doc_id"`
		DocInfo         string        `json:"doc_info"`
		DocName         string        `json:"doc_name"`
		HosId           string        `json:"hos_id"`
		HosName         string        `json:"hos_name"`
		HospitalType    int64         `json:"hospital_type"`
		HosType         int64         `json:"hos_type"`
		IsPage          int64         `json:"is_page"`
		LevelName       string        `json:"level_name"`
		ReservationType int64         `json:"reservation_type"`
		HospitalRule    string        `json:"hospital_rule"`
		IsDatepart      int64         `json:"is_datepart"`
		Favorite        int64         `json:"favorite"`
		StopInfo        string        `json:"stop_info"`
		IsInnerSystem   int64         `json:"is_inner_system"`
	} `json:"data"`
}

type hpvScheduleResp struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ScheduleId int64  `json:"schedule_id"`
		TimeType   string `json:"time_type"`
		SchDate    string `json:"sch_date"`
		SrcMax     int64  `json:"src_max"`
		SrcNum     int64  `json:"src_num"`
		CateName   string `json:"cate_name"`
		Ghf        int64  `json:"ghf"`
		Zlf        int64  `json:"zlf"`
		Zjf        int64  `json:"zjf"`
		Amt        int64  `json:"amt"`
		DocId      string `json:"doc_id"`
		IsDatepart int64  `json:"is_datepart"`
	} `json:"data"`
}

type mailInfo struct {
	SchDate  string
	CateName string
	docName  string
	hosName  string
	SrcMax   int64
	SrcNum   int64
}

type appointResp struct {
	Code int64         `json:"code"`
	Msg  string        `json:"msg"`
	Data []interface{} `json:"data"`
}

type appointBody struct {
	DocName     string `json:"doc_name"`
	HosName     string `json:"hos_name"`
	HosCode     string `json:"hos_code"`
	DepName     string `json:"dep_name"`
	LevelName   string `json:"level_name"`
	DepId       string `json:"dep_id"`
	DocId       string `json:"doc_id"`
	PatId       int64  `json:"pat_id"`
	ScheduleId  int64  `json:"schedule_id"`
	JzCard      string `json:"jz_card"`
	SchDate     string `json:"sch_date"`
	TimeType    string `json:"time_type"`
	Info        string `json:"info"`
	Ghf         int64  `json:"ghf"`
	Zlf         int64  `json:"zlf"`
	Zjf         int64  `json:"zjf"`
	JzStartTime int64  `json:"jz_start_time"`
	Amt         int64  `json:"amt"`
	JzCardType  int64  `json:"jz_card_type"`
	QueueSnId   string `json:"queue_sn_id"`
	WechatLogin string `json:"wechat_login"`
}

type swapData struct {
	hosName string
	docName string
	docGood string
	hosId   string
	docId   string
	depId   string
	depName string
}

var wg sync.WaitGroup

func main() {
	if !IsDebug {
		// Release
		errorCount := ErrorCount

	Release:
		// //////////START//////////
		fmt.Printf("↓==========%v==========↓\n", time.Now().Format("2006-01-02 15:04:05"))

		h, err := getHosList()
		if err != nil {
			errorCount--
			fmt.Println(err.Error())
			if errorCount > 0 {
				goto Release
			} else {
				panic(err)
			}
		}

		for i := 0; i < ThreadNum; i++ {
			wg.Add(1)
			go loopThread(h, i)
		}

		wg.Wait()

		fmt.Printf("↑==========%v==========↑\n", time.Now().Format("2006-01-02 15:04:05"))

		// //////////END//////////
	} else {
		// Debug

	}

	return
}

func loopThread(h hosResp, num int) (err error) {
	defer wg.Done()

	for i, d := range h.Data {
		var swaps []swapData

		if i%ThreadNum == num {
			for _, doctor := range d.Doctor {
				// Catch all hpv programme
				var swap swapData
				swap, _, err = getHosDetail(strconv.FormatInt(doctor.DocId, 10), strconv.FormatInt(d.HosCode, 10), strconv.FormatInt(doctor.DepId, 10))
				swaps = append(swaps, swap)

				if err != nil {
					fmt.Println(err.Error())
					continue
				}

				// Catch hpv remaining
				_, _, err = getHpvSchedule(&swaps, strconv.FormatInt(doctor.DocId, 10), strconv.FormatInt(d.HosCode, 10), strconv.FormatInt(doctor.DepId, 10))

				if err != nil {
					fmt.Println(err.Error())
					continue
				}
			}
		}
	}
	return nil
}

func getHosList() (h hosResp, err error) {
	client := resty.New()
	resp, err := client.R().Get(DytApiHost + "Vaccine/hpvhoslist")
	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &h)
	if err != nil {
		return
	}

	return
}

func getHosDetail(docId string, hosCode string, depId string) (swap swapData, hd hosDetailResp, err error) {
	client := resty.New()
	resp, err := client.R().
		SetHeaders(map[string]string{
			"Host":            "newdytapi.ynhdkc.com",
			"Origin":          "https://appv2.ynhdkc.com",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection":      "keep-alive",
			"Accept":          "application/json, text/plain, */*",
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) MicroMessenger/6.8.0(0x16080000) MacWechat/3.4(0x13040010) MiniProgramEnv/Mac MiniProgram",
			"Referer":         "https://appv2.ynhdkc.com/",
			"Accept-Language": "zh-CN,zh-Hans;q=0.9",
			"Content-Type":    "text/plain",
		}).
		Get(DytApiHost + "index/doctor/" + docId + "?hos_code=" + hosCode + "&dep_id=" + depId + "&vip=0")
	// Such as: 2713, 872003, 752

	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &hd)

	if err != nil {
		return
	}

	swap = swapData{hd.Data.HosName, hd.Data.DocName, hd.Data.DocGood, hd.Data.HosId, hd.Data.DocId, hd.Data.DepId, hd.Data.DepName}

	// fmt.Printf(time.Now().Format("2006-01-02 15:04:05")+"%v:\t%v\n", hd.Data.HosName, hd.Data.DocName)
	return
}

func getHpvSchedule(swaps *[]swapData, docId string, hosCode string, depId string) (hs hpvScheduleResp, str string, err error) {
	client := resty.New()
	resp, err := client.R().
		SetHeaders(map[string]string{
			"Host":            "newdytapi.ynhdkc.com",
			"Origin":          "https://appv2.ynhdkc.com",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection":      "keep-alive",
			"Accept":          "application/json, text/plain, */*",
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) MicroMessenger/6.8.0(0x16080000) MacWechat/3.4(0x13040010) MiniProgramEnv/Mac MiniProgram",
			"Referer":         "https://appv2.ynhdkc.com/",
			"Accept-Language": "zh-CN,zh-Hans;q=0.9",
			"Content-Type":    "text/plain",
		}).
		Get(DytApiHost + "index/schedule?hos_code=" + hosCode + "&dep_id=" + depId + "&doc_id=" + docId + "&hyid=&vip=0")

	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &hs)

	if err != nil {
		return
	}

	for _, d := range hs.Data {
		var (
			hosName string
			docName string
			// docGood string
			// hosId   string
			docId   string
			depId   string
			depName string
		)

		for _, swap := range *swaps {
			if swap.docId == d.DocId {
				hosName = swap.hosName
				docName = swap.docName
				// docGood = swap.docGood
				// hosId = swap.hosId
				docId = swap.docId
				depId = swap.depId
				depName = swap.depName
			}
		}

		str = fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v", d.SchDate, d.CateName, docName, hosName, d.SrcMax, d.SrcNum)

		if strings.Contains(docName, KeyWord) {
			// Log
			fmt.Println(time.Now().Format("2006-01-02 15:04:05") + "+\t" + str)
		}

		if d.SrcNum > 0 && strings.Contains(docName, KeyWord) {
			// Send Email
			if IsSending {
				err := sendEmail("[滇医通]HPV疫苗余量提示",
					fmt.Sprintf("时间：%v\t%v\n地点：%v\n项目：%v\n计划：%v\n剩余：%v", d.SchDate, d.CateName, docName, hosName, d.SrcMax, d.SrcNum),
				)
				if err != nil {
					fmt.Println(err.Error())
				} else {
					fmt.Println("Sending Successfully")
				}
			}

			fmt.Println("Remaining...")

			// Appointment
			appointCount := AppointCount

		DoAppoint:
			if IsAppointment {
				fmt.Println("DoAppoint...")

				appointCount--
				var aResp appointResp
				aResp, err = appointHpv(hosCode, depId, docId, PatId, UserId, strconv.FormatInt(d.ScheduleId, 10), "",
					docName, hosName, depName, d.SchDate, d.TimeType,
				)
				if err != nil {
					return
				}

				aRespMsg := aResp.Msg

				fmt.Println("!!!!!!!!!!!!!!!!!!!!")
				fmt.Println(aResp)
				fmt.Println("!!!!!!!!!!!!!!!!!!!!")

				if strings.Contains(aRespMsg, "成功") {
					// Success
					fmt.Println("Appoint Successfully")

					if IsSending {
						err := sendEmail("[滇医通]HPV自动预约成功",
							fmt.Sprintf("时间：%v\t%v\n地点：%v\n项目：%v\n计划：%v\n剩余：%v", d.SchDate, d.CateName, docName, hosName, d.SrcMax, d.SrcNum),
						)
						if err != nil {
							fmt.Println(err.Error())
						} else {
							fmt.Println("Sending Successfully")
						}
					}

					return
				} else if strings.Contains(aRespMsg, "失败") {
					fmt.Println("Appoint Unsuccessfully")

					// ReAppoint

					fmt.Println("ReAppoint...")
					if appointCount > 0 {
						time.Sleep(AppointSleep * time.Millisecond)
						goto DoAppoint
					} else {
						return
					}
				} else if strings.Contains(aRespMsg, "被抢空") {
					// Next programme
					fmt.Println("0 Remaining, End This Programme...")
					return
				} else {
					// Sending abnormal message
					fmt.Println("Response Msg Error: " + aRespMsg)

					if IsSending {
						err := sendEmail("[滇医通]结果返回异常",
							fmt.Sprintf("%v", aRespMsg),
						)
						if err != nil {
							fmt.Println(err.Error())
						} else {
							fmt.Println("Sending Successfully")
						}
					}

					// ReAppoint
					fmt.Println("ReAppoint...")
					if appointCount > 0 {
						time.Sleep(AppointSleep * time.Millisecond)
						goto DoAppoint
					} else {
						return
					}
				}
			}
		}
	}

	return
}

func appointHpv(
	hosCode string,
	depId string,
	docId string,
	patId string,
	userId string,
	scheduleId string,
	cateName string,

	docName string,
	hosName string,
	depName string,
	schDate string,
	timeType string,
) (aResp appointResp, err error) {
	postBody := fmt.Sprintf(`
{
  "doc_name": "%v",
  "hos_name": "%v",
  "hos_code": "%v",
  "dep_name": "%v",
  "level_name": "",
  "dep_id": "%v",
  "doc_id": "%v",
  "pat_id": %v,
  "schedule_id": %v,
  "jz_card": "",
  "sch_date": "%v",
  "time_type": "%v",
  "info": "",
  "ghf": 0,
  "zlf": 0,
  "zjf": 0,
  "jz_start_time": 0,
  "amt": 0,
  "jz_card_type": 0,
  "queue_sn_id": "",
  "wechat_login": "dytminiapp"
}
`, docName, hosName, hosCode, depName, depId, docId, patId, scheduleId, schDate, timeType)

	client := resty.New()
	resp, err := client.R().
		SetHeaders(map[string]string{
			"Host":            "dytapi.ynhdkc.com",
			"Origin":          "https://appv2.ynhdkc.com",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection":      "keep-alive",
			"Accept":          "application/json, text/plain, */*",
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) MicroMessenger/6.8.0(0x16080000) MacWechat/3.4(0x13040010) MiniProgramEnv/Mac MiniProgram",
			"Referer":         "https://appv2.ynhdkc.com/",
			"Accept-Language": "zh-CN,zh-Hans;q=0.9",
			"Content-Type":    "application/json",

			"x-uuid":        XUuid,
			"Authorization": Authorization,
		}).
		SetQueryParams(map[string]string{
			"hos_code":    hosCode,
			"dep_id":      depId,
			"doc_id":      docId,
			"pat_id":      patId,
			"user_id":     userId,
			"schedule_id": scheduleId,
			"cate_name":   cateName,
		}).
		SetBody(postBody).
		Post(DytApiHost + "v1/appoint")

	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &aResp)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	return
}

func sendEmail(subject string, text string) (err error) {
	e := email.NewEmail()
	e.From = EmailUser
	e.To = []string{EmailTo1}
	if EmailTo2 != "" {
		e.To = append(e.To, EmailTo2)
	}
	e.Subject = subject
	e.Text = []byte(text)
	// 25 port is blocked on Aliyun
	err = e.Send("smtp.88.com:25", smtp.PlainAuth("", EmailUser, EmailPass, "smtp.88.com"))

	if err != nil {
		return
	}

	return
}
