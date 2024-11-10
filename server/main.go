package main

import (
	"bilidown/router"
	"bilidown/util"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"
	_ "modernc.org/sqlite"
)

const (
	HTTP_PORT = 8098
	HTTP_HOST = ""
	VERSION   = "v2.0.8"
)

var urlLocal = fmt.Sprintf("http://127.0.0.1:%d", HTTP_PORT)
var urlLocalUnix = fmt.Sprintf("%s?___%d", urlLocal, time.Now().UnixMilli())

func main() {
	checkFFmpeg()
	systray.Run(onReady, nil)
}

// checkFFmpeg 检测 ffmpeg 的安装情况，如果未安装则打印提示信息。
func checkFFmpeg() {
	if _, err := util.GetFFmpegPath(); err != nil {
		fmt.Println("🚨 FFmpeg is missing. Install it from https://www.ffmpeg.org/download.html or place it in ./bin, then restart the application.")
		var wg sync.WaitGroup
		wg.Add(1)
		wg.Wait()
	}
}

func onReady() {
	setIcon()
	setTitle()
	setMenuItem()
	initTables()
	setServer()
	time.Sleep(time.Millisecond * 1000)
	openBrowser(urlLocalUnix)
	keepWait()
}

// keepWait 阻塞终端
func keepWait() {
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

func setServer() {
	// 前端打包文件
	http.Handle("/", http.FileServer(http.Dir("static")))
	// 后端接口服务
	http.Handle("/api/", http.StripPrefix("/api", router.API()))
	// 启动 HTTP 服务器
	go func() {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", HTTP_HOST, HTTP_PORT), nil)
		if err != nil {
			log.Fatal("http.ListenAndServe:", err)
		}
	}()
}

func openBrowser(_url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", _url)
	case "darwin":
		cmd = exec.Command("open", _url)
	case "linux":
		cmd = exec.Command("xdg-open", _url)
	default:
		log.Printf("openBrowser: %v.", errors.New("unsupported operating system"))
	}
	if err := cmd.Start(); err != nil {
		log.Printf("openBrowser: %v.", err)
	}
	fmt.Printf("Opened in default browser: %s.\n", _url)
}

func setIcon() {
	var path string
	if runtime.GOOS == "windows" {
		path = "static/favicon.ico"
	} else {
		path = "static/favicon-32x32.png"
	}
	systray.SetIcon(mustReadFile(path))
}

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln("os.ReadFile:", err)
	}
	return data
}

func setTitle() {
	title := "Bilidown"
	tooltip := fmt.Sprintf("%s 视频解析器 %s (port:%d)", title, VERSION, HTTP_PORT)
	// only available on Mac and Linux.
	systray.SetTitle(title)
	// only available on Mac and Windows.
	systray.SetTooltip(tooltip)
}

func setMenuItem() {
	openBrowserItemText := fmt.Sprintf("打开主界面 (port:%d)", HTTP_PORT)
	openBrowserItem := systray.AddMenuItem(openBrowserItemText, openBrowserItemText)
	go func() {
		for {
			<-openBrowserItem.ClickedCh
			openBrowser(urlLocalUnix)
		}
	}()

	aboutItemText := "Github 项目主页"
	aboutItem := systray.AddMenuItem(aboutItemText, aboutItemText)
	go func() {
		for {
			<-aboutItem.ClickedCh
			openBrowser("https://github.com/iuroc/bilidown")
		}
	}()

	exitItemText := "退出应用"
	exitItem := systray.AddMenuItem(exitItemText, exitItemText)
	go func() {
		<-exitItem.ClickedCh
		log.Printf("Bilidown has exited.")
		systray.Quit()
	}()
}

// initTables 初始化数据表
func initTables() {
	db := util.MustGetDB()
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS "field" (
		"name" TEXT PRIMARY KEY NOT NULL,
		"value" TEXT
	)`); err != nil {
		log.Fatalln("create table field:", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS "log" (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"content" TEXT NOT NULL,
		"create_at" text NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		log.Fatalln("create table log:", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS "task" (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"bvid" text NOT NULL,
		"cid" integer NOT NULL,
		"format" integer NOT NULL,
		"title" text NOT NULL,
		"owner" text NOT NULL,
		"cover" text NOT NULL,
		"status" text NOT NULL,
		"folder" text NOT NULL,
		"duration" integer NOT NULL,
		"create_at" text NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		log.Fatalln("create table task:", err)
	}

	if _, err := util.GetCurrentFolder(db); err != nil {
		log.Fatalln("util.GetCurrentFolder:", err)
	}

	if err := initHistoryTask(db); err != nil {
		log.Fatalln("initHistoryTask:", err)
	}
}

// initHistoryTask 将上一次程序运行时未完成的任务进度全部变为 error
func initHistoryTask(db *sql.DB) error {
	util.SqliteLock.Lock()
	_, err := db.Exec(`UPDATE "task" SET "status" = 'error' WHERE "status" IN ('waiting', 'running')`)
	util.SqliteLock.Unlock()
	return err
}
