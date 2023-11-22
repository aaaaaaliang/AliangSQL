package main

import (
	"awesomeProject4/storgeengine"
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// 创建数据库实例
	db := storgeengine.NewDB()

	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("启动服务出错: ", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("已启动localhost:8080监听")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("连接出错: ", err)
			continue
		}
		go handleRequest(conn, db)
	}
}

func handleRequest(conn net.Conn, db *storgeengine.DB) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(conn, "读取命令出错: %s\n", err)
			break
		}

		message = strings.TrimSpace(message)
		fmt.Println("接收到的命令:", message)
		if strings.ToLower(message) == "exit;" {
			fmt.Fprintf(conn, "退出命令接收\n")
			break
		}

		result := storgeengine.ParseSQL(message, db)
		if result.Error != nil {
			fmt.Fprintf(conn, "执行命令出错: %s\n", result.Error)
		} else {
			if result.Result != nil {
				fmt.Fprintf(conn, "%s\n", result.Result)
			} else {
				fmt.Fprintf(conn, "命令执行成功\n")
			}
		}
		// 添加结束标记
		fmt.Fprintf(conn, "END\n")
	}
	fmt.Println("handleRequest函数结束，连接关闭")
}
