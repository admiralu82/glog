// go build -ldflags -H=windowsgui .
package main

import (
	"fmt"
	"gordp/wrappers"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	LOGIN   = "log"
	LOGOUT  = "out"
	CSVSEP  = ";"
	LOGFILE = "info.csv"
)

func main() {
	// logStatus := LOGIN
	// if len(os.Args) > 0 {
	// 	if strings.Contains(os.Args[0], "_on") {
	// 		logStatus = LOGIN
	// 	}
	// 	if strings.Contains(os.Args[0], "_off") {
	// 		logStatus = LOGOUT
	// 	}
	// }
	// if len(os.Args) > 1 {
	// 	if os.Args[1] == "on" {
	// 		logStatus = LOGIN
	// 	}
	// 	if os.Args[1] == "off" {
	// 		logStatus = LOGOUT
	// 	}
	// }

	sessionName := strings.ToUpper(os.Getenv("SESSIONNAME"))
	userName := os.Getenv("USERNAME")
	nowTime := time.Now().Format("15:04:05")
	nowDate := time.Now().Format("2006-01-02")

	var ipAddr, srcCompName string
	var err error
	if strings.Contains(sessionName, "CONSOLE") {
		ipAddr = "127.0.0.1"
		srcCompName = "localcomp"

	} else {
		retBytes := uint32(0)
		buffer := make([]byte, 10000)

		// запрос на IP адрес
		err = wrappers.WTSQuerySessionInformation(0, 0xFFFFFFFF, wrappers.WTSClientAddress, (**uint16)(unsafe.Pointer(&buffer)), &retBytes)
		if err == nil {
			if buffer[0] == 2 {
				ipAddr = strconv.Itoa(int(buffer[6])) + "." + strconv.Itoa(int(buffer[7])) + "." + strconv.Itoa(int(buffer[8])) + "." + strconv.Itoa(int(buffer[9]))
			}
		} else {
			ipAddr = err.Error()
		}

		// userName := ""
		err = wrappers.WTSQuerySessionInformation(0, 0xFFFFFFFF, wrappers.WTSClientInfo, (**uint16)(unsafe.Pointer(&buffer)), &retBytes)
		if err == nil && retBytes > 42 {
			tmpBuf := make([]uint16, 50)
			for i := range buffer {
				if i%2 == 1 {
					val := uint16(buffer[i])*256 + uint16(buffer[i-1])
					tmpBuf[(i-1)/2] = val

					if val == 0 {
						break
					}
				}
			}
			srcCompName = UTF16ToString(tmpBuf)
		} else {
			srcCompName = err.Error()
		}

		mem := (*byte)(unsafe.Pointer(&buffer))
		wrappers.WTSFreeMemory(mem)
	}

	s := CSVSEP
	outInfo := sessionName + s + userName + s + ipAddr + s + srcCompName + s + GetGroups()

	loginMsg := LOGIN + s + nowDate + s + nowTime + s + outInfo
	fmt.Println(loginMsg)
	WriteFile(loginMsg)

	// ждем выхода
	signalChan := make(chan os.Signal, 1)
	nowDate = time.Now().Format("2006-01-02")
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan

		nowTime = time.Now().Format("15:04:05")
		nowDate = time.Now().Format("2006-01-02")

		logoutMsg := LOGOUT + s + nowDate + s + nowTime + s + outInfo
		fmt.Println(logoutMsg)
		WriteFile(logoutMsg)

		os.Exit(0)
	}()

	for {
		time.Sleep(time.Hour)
	}
}

func UTF16ToString(s []uint16) string {
	for i, v := range s {
		if v == 0 {
			s = s[0:i]
			break
		}
	}
	return string(utf16.Decode(s))
}

func WriteFile(text string) {

	logFile := os.Args[0]
	pos := strings.LastIndex(logFile, "\\")
	if pos == -1 {
		logFile = LOGFILE
	} else {
		logFile = string([]byte(logFile)[:pos+1])
		logFile = logFile + LOGFILE
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	if _, err = f.WriteString(text + "\r\n"); err != nil {
		fmt.Println(err)
		return
	}
}

func GetGroups() string {
	u, err := user.Current()
	if err != nil {
		fmt.Println("current_user error", err)
		return err.Error()
	}

	g, err := u.GroupIds()
	if err != nil {
		fmt.Println("current_group error", err)
		return err.Error()
	}

	gName := []string{}
	for i := range g {
		ug, err := user.LookupGroupId(g[i])
		if err != nil {
			fmt.Println("get_group error", err)
			continue
		}
		if strings.Compare("Отсутствует", ug.Name) == 0 {
			continue
		}

		gName = append(gName, ug.Name)
	}

	out := ""
	for i := range gName {
		add := ""
		if i > 0 {
			add = ","
		}

		out = out + add + gName[i]
	}
	return out
}
