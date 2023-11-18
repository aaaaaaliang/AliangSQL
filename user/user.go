package user

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Welcome() {
	fmt.Println("**************************************************************")
	fmt.Println("*                                                            *")
	fmt.Println("*                    欢迎来到liangdb                         *")
	fmt.Println("*                                                            *")
	fmt.Println("**************************************************************")
	fmt.Println("*                      操作指令                              *")
	fmt.Println("*                输入help;获取帮助指令                       *")
	fmt.Println("*                    SQL语句需要;结尾                        *")
	fmt.Println("*            如遇到输入字符串请用单引号 ('') !!!!            *")
	fmt.Println("**************************************************************")
}

func UserInput() string {
	reader := bufio.NewReader(os.Stdin)
	sqlCommand := ""

	fmt.Println("请输入 SQL 命令，以分号 ; 结尾")

	for {
		fmt.Print("SQL> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时发生错误:", err)
			break
		}

		sqlCommand += input

		if strings.Contains(sqlCommand, ";") {
			sqlCommand = strings.ToUpper(sqlCommand)
			break // 结束循环
		}
	}
	sql := ""
	for _, char := range sqlCommand {
		if char != ';' {
			sql += string(char)
		}
	}
	return sql // 返回用户输入的 SQL 命令
}

func UserLogin() {
	filePath := "/Users/zhangxueliang/GolandProjects/liangsql/data/users.txt"
	userDB, err := InitializeUserDB(filePath)
	if err != nil {
		fmt.Println("无法初始化用户数据库:", err)
		return
	}
Login:
	for {
		// 提示用户输入用户名和密码
		var username, password string
		fmt.Print("用户名: ")
		fmt.Scanln(&username)
		fmt.Print("密码: ")
		fmt.Scanln(&password)
		// 尝试用户登录
		err = Login(userDB, username, password)
		if err != nil {
			fmt.Println("登录失败:", err)
			continue Login
		}
		fmt.Printf("登录成功，欢迎 %s!\n", username)
		break
	}
}

func InitializeUserDB(filePath string) (map[string]string, error) {
	userDB := make(map[string]string)
	file, err := openOrCreateUserFile(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// 建一个文本扫描器，逐行扫描文本数据
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			username, password := parts[0], parts[1]
			userDB[username] = password
		}
	}

	return userDB, nil
}

// 打开用户数据文件，如果文件不存在则创建
func openOrCreateUserFile(filePath string) (*os.File, error) {
	// 判断是否存在文件
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		file, createErr := os.Create(filePath)
		if createErr != nil {
			return nil, createErr
		}
		userAccount := "root:1234"
		_, err := file.WriteString(userAccount)
		if err != nil {
			fmt.Println("写入密码出错")
			return nil, err
		}
		return file, nil
	}

	file, openErr := os.Open(filePath)
	if openErr != nil {
		return nil, openErr
	}
	return file, nil
}

func Login(userDB map[string]string, username, password string) error {
	storedPassword, userExists := userDB[username]
	if !userExists || storedPassword != password {
		return fmt.Errorf("用户名或密码不正确")
	}
	return nil
}
