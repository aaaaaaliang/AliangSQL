package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("dial连接出错: ", err)
		os.Exit(1)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("请输入 SQL 命令，以分号 ; 结尾: ")
		sqlCommand, _ := reader.ReadString('\n')
		sqlCommand = strings.TrimSpace(sqlCommand)

		if !strings.HasSuffix(sqlCommand, ";") {
			fmt.Println("SQL 命令必须以分号 ';' 结尾，请重新输入。")
			continue
		}

		if strings.ToLower(sqlCommand) == "exit;" {
			conn.Write([]byte(sqlCommand + "\n"))
			break
		}

		_, err := conn.Write([]byte(sqlCommand + "\n"))
		if err != nil {
			fmt.Println("client发送出错: ", err)
			continue
		}

		response := handleResponse(conn)
		fmt.Println("来自服务器的响应:\n" + response)
	}
}

func handleResponse(conn net.Conn) string {
	var response strings.Builder
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取服务器响应出错: ", err)
			break
		}

		if strings.TrimSpace(line) == "END" {
			break
		}
		response.WriteString(line)
	}
	return response.String()
}
