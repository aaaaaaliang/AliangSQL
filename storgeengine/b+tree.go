package storgeengine

import (
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type BPItem struct {
	Key int64
	Val map[string]interface{}
}

// BPNode b+tree节点
type BPNode struct {
	MaxKey int64     // 最大关键字
	Nodes  []*BPNode // 子节点
	Items  []BPItem  // 子数据项
	Next   *BPNode   // 指针
}

// 查找数据项，返回子数据项的索引
func (node *BPNode) findItem(key int64) int {
	num := len(node.Items)
	for i := 0; i < num; i++ {
		if node.Items[i].Key > key {
			return -1
		} else if node.Items[i].Key == key {
			return i
		}
	}
	return -1
}

// 为item赋值
func (node *BPNode) setValue(key int64, value map[string]interface{}) {
	item := BPItem{key, value}
	num := len(node.Items)
	// 保证插入的位置有序
	if num < 1 {
		node.Items = append(node.Items, item)
		node.MaxKey = item.Key
		return
	} else if key < node.Items[0].Key {
		node.Items = append([]BPItem{item}, node.Items...)
		return
	} else if key > node.Items[num-1].Key {
		node.Items = append(node.Items, item)
		node.MaxKey = item.Key
		return
	}

	for i := 0; i < num; i++ {
		if node.Items[i].Key > key {
			node.Items = append(node.Items, BPItem{})
			copy(node.Items[i+1:], node.Items[i:])
			node.Items[i] = item
			return
		} else if node.Items[i].Key == key {
			node.Items[i] = item
			return
		}
	}
}

// 插入子节点，保证子节点有序
func (node *BPNode) addChild(child *BPNode) {
	num := len(node.Nodes)
	if num < 1 {
		node.Nodes = append(node.Nodes, child)
		node.MaxKey = child.MaxKey
		return
	} else if child.MaxKey < node.Nodes[0].MaxKey {
		node.Nodes = append([]*BPNode{child}, node.Nodes...)
		return
	} else if child.MaxKey > node.Nodes[num-1].MaxKey {
		node.Nodes = append(node.Nodes, child)
		node.MaxKey = child.MaxKey
		return
	}

	for i := 0; i < num; i++ {
		if node.Nodes[i].MaxKey > child.MaxKey {
			node.Nodes = append(node.Nodes, nil)
			copy(node.Nodes[i+1:], node.Nodes[i:])
			node.Nodes[i] = child
			return
		}
	}
}

// 删除子数据项
func (node *BPNode) deleteItem(key int64) bool {
	num := len(node.Items)
	for i := 0; i < num; i++ {
		if node.Items[i].Key > key {
			return false
		} else if node.Items[i].Key == key {
			copy(node.Items[i:], node.Items[i+1:])
			node.Items = node.Items[0 : len(node.Items)-1]
			node.MaxKey = node.Items[len(node.Items)-1].Key
			return true
		}
	}
	return false
}

// 删除子节点
func (node *BPNode) deleteChild(child *BPNode) bool {
	num := len(node.Nodes)
	for i := 0; i < num; i++ {
		if node.Nodes[i] == child {
			copy(node.Nodes[i:], node.Nodes[i+1:])
			node.Nodes = node.Nodes[0 : len(node.Nodes)-1]
			if len(node.Nodes) > 0 {
				node.MaxKey = node.Nodes[len(node.Nodes)-1].MaxKey
			}
			return true
		}
	}
	return false
}

// BPTree 整体的BPTree结构
type BPTree struct {
	mutex sync.RWMutex // 锁
	root  *BPNode
	width int // B+树的宽度
	halfw int
	table *BPTable // 存储表的结构信息
}

func NewBPTree(width int) *BPTree {
	if width < 3 {
		width = 3
	}
	var bt = &BPTree{}
	bt.root = NewLeafNode(width)
	bt.width = width
	bt.halfw = (bt.width + 1) / 2 //分裂条件 保证b+树的平衡
	return bt
}

// NewLeafNode 申请width+1是因为插入时可能暂时出现节点key大于申请width的情况,待后期再分裂处理
func NewLeafNode(width int) *BPNode {
	var node = &BPNode{}
	node.Items = make([]BPItem, width+1)
	node.Items = node.Items[0:0]
	return node
}

// NewIndexNode 申请width+1是因为插入时可能暂时出现节点key大于申请width的情况,待后期再分裂处理
func NewIndexNode(width int) *BPNode {
	var node = &BPNode{}
	node.Nodes = make([]*BPNode, width+1)
	node.Nodes = node.Nodes[0:0]
	return node
}

// Get 从根节点一步一步向下遍历，找到key对应的值
func (t *BPTree) Get(key int64) interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	node := t.root
	for i := 0; i < len(node.Nodes); i++ {
		if key <= node.Nodes[i].MaxKey {
			node = node.Nodes[i]
			i = 0
		}
	}

	//没有到达叶子结点
	if len(node.Nodes) > 0 {
		return nil
	}

	for i := 0; i < len(node.Items); i++ {
		if node.Items[i].Key == key {
			return node.Items[i].Val
		}
	}
	return nil
}

func (db *DB) SelectAll(tableName string) map[int64]interface{} {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	table, exists := db.databases[db.currentDB][tableName]
	if !exists {
		fmt.Printf("表 %s 不存在\n", tableName)
		return nil
	}
	return table.Tree.getAllData()
}

func (t *BPTree) getAllData() map[int64]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.getData(t.root)
}

func (t *BPTree) getData(node *BPNode) map[int64]interface{} {
	data := make(map[int64]interface{})
	for i := 0; i < len(node.Items); i++ {
		data[node.Items[i].Key] = node.Items[i].Val
	}

	for i := 0; i < len(node.Nodes); i++ {
		subData := t.getData(node.Nodes[i])
		for key, val := range subData {
			data[key] = val
		}
	}
	return data
}

func (t *BPTree) GetData() map[int64]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.getData(t.root)
}

// 分裂操作
func (t *BPTree) splitNode(node *BPNode) *BPNode {
	if len(node.Nodes) > t.width {
		//创建新结点
		halfw := t.width/2 + 1
		node2 := NewIndexNode(t.width)
		node2.Nodes = append(node2.Nodes, node.Nodes[halfw:len(node.Nodes)]...)
		node2.MaxKey = node2.Nodes[len(node2.Nodes)-1].MaxKey

		//修改原结点数据
		node.Nodes = node.Nodes[0:halfw]
		node.MaxKey = node.Nodes[len(node.Nodes)-1].MaxKey

		return node2
	} else if len(node.Items) > t.width {
		//创建新结点
		halfw := t.width/2 + 1
		node2 := NewLeafNode(t.width)
		node2.Items = append(node2.Items, node.Items[halfw:len(node.Items)]...)
		node2.MaxKey = node2.Items[len(node2.Items)-1].Key

		//修改原结点数据
		node.Next = node2
		node.Items = node.Items[0:halfw]
		node.MaxKey = node.Items[len(node.Items)-1].Key

		return node2
	}

	return nil
}

func (t *BPTree) setValue(parent *BPNode, node *BPNode, key int64, value map[string]interface{}) {
	for i := 0; i < len(node.Nodes); i++ {
		if key <= node.Nodes[i].MaxKey || i == len(node.Nodes)-1 {
			t.setValue(node, node.Nodes[i], key, value)
			break
		}
	}

	//叶子结点，添加数据
	if len(node.Nodes) < 1 {
		node.setValue(key, value)
	}

	//结点分裂
	node_new := t.splitNode(node)
	if node_new != nil {
		//节点为根节点的情况
		if parent == nil {
			parent = NewIndexNode(t.width)
			parent.addChild(node)
			t.root = parent
		}

		parent.addChild(node_new)
	}
}

func (t *BPTree) Set(key int64, value map[string]interface{}) { // 修改这里
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.setValue(nil, t.root, key, value)
}

func (t *BPTree) itemMoveOrMerge(parent *BPNode, node *BPNode) {
	//获取兄弟结点
	var node1 *BPNode = nil
	var node2 *BPNode = nil
	if parent != nil {
		for i := 0; i < len(parent.Nodes); i++ {
			if parent.Nodes[i] == node {
				if i < len(parent.Nodes)-1 {
					node2 = parent.Nodes[i+1]
				} else if i > 0 {
					node1 = parent.Nodes[i-1]
				}
				break
			}
		}
	}

	//将左侧结点的记录移动到删除结点
	if node1 != nil && len(node1.Items) > t.halfw {
		item := node1.Items[len(node1.Items)-1]
		node1.Items = node1.Items[0 : len(node1.Items)-1]
		node1.MaxKey = node1.Items[len(node1.Items)-1].Key
		node.Items = append([]BPItem{item}, node.Items...)
		return
	}

	//将右侧结点的记录移动到删除结点
	if node2 != nil && len(node2.Items) > t.halfw {
		item := node2.Items[0]
		node2.Items = node2.Items[1:] // 修正此行
		node.Items = append(node.Items, item)
		node.MaxKey = node.Items[len(node.Items)-1].Key
		return
	}

	//与左侧结点进行合并
	if node1 != nil && len(node1.Items)+len(node.Items) <= t.width {
		node1.Items = append(node1.Items, node.Items...)
		node1.Next = node.Next
		node1.MaxKey = node1.Items[len(node1.Items)-1].Key
		if parent != nil {
			parent.deleteChild(node)
		}
		return
	}

	//与右侧结点进行合并
	if node2 != nil && len(node2.Items)+len(node.Items) <= t.width {
		node.Items = append(node.Items, node2.Items...)
		node.Next = node2.Next
		node.MaxKey = node.Items[len(node.Items)-1].Key
		if parent != nil {
			parent.deleteChild(node2)
		}
		return
	}
}

func (t *BPTree) childMoveOrMerge(parent *BPNode, node *BPNode) {
	if parent == nil {
		return
	}

	//获取兄弟结点
	var node1 *BPNode = nil
	var node2 *BPNode = nil
	for i := 0; i < len(parent.Nodes); i++ {
		if parent.Nodes[i] == node {
			if i < len(parent.Nodes)-1 {
				node2 = parent.Nodes[i+1]
			} else if i > 0 {
				node1 = parent.Nodes[i-1]
			}
			break
		}
	}

	//将左侧结点的子结点移动到删除结点
	if node1 != nil && len(node1.Nodes) > t.halfw {
		item := node1.Nodes[len(node1.Nodes)-1]
		node1.Nodes = node1.Nodes[0 : len(node1.Nodes)-1]
		node.Nodes = append([]*BPNode{item}, node.Nodes...)
		return
	}

	//将右侧结点的子结点移动到删除结点
	if node2 != nil && len(node2.Nodes) > t.halfw {
		item := node2.Nodes[0]
		node2.Nodes = node1.Nodes[1:]
		node.Nodes = append(node.Nodes, item)
		return
	}

	if node1 != nil && len(node1.Nodes)+len(node.Nodes) <= t.width {
		node1.Nodes = append(node1.Nodes, node.Nodes...)
		parent.deleteChild(node)
		return
	}

	if node2 != nil && len(node2.Nodes)+len(node.Nodes) <= t.width {
		node.Nodes = append(node.Nodes, node2.Nodes...)
		parent.deleteChild(node2)
		return
	}
}

func (t *BPTree) deleteItem(parent *BPNode, node *BPNode, key int64) {
	for i := 0; i < len(node.Nodes); i++ {
		if key <= node.Nodes[i].MaxKey {
			t.deleteItem(node, node.Nodes[i], key)
			break
		}
	}

	if len(node.Nodes) < 1 {
		//删除记录后若结点的子项<m/2，则从兄弟结点移动记录，或者合并结点
		node.deleteItem(key)
		if len(node.Items) < t.halfw {
			t.itemMoveOrMerge(parent, node)
		}
	} else {
		//若结点的子项<m/2，则从兄弟结点移动记录，或者合并结点
		node.MaxKey = node.Nodes[len(node.Nodes)-1].MaxKey
		if len(node.Nodes) < t.halfw {
			t.childMoveOrMerge(parent, node)
		}
	}
}

func (t *BPTree) Remove(key int64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.deleteItem(nil, t.root, key)
}
func (t *BPTree) Insert(key int64, value map[string]interface{}) {
	t.Set(key, value)
}

// Select 查询数据
func (t *BPTree) Select(key int64) (interface{}, bool) {
	value := t.Get(key)
	if value != nil {
		return value, true
	}
	return nil, false
}

// Delete 删除数据
func (t *BPTree) Delete(key int64) bool {
	t.Remove(key)
	_, exists := t.Select(key)
	return !exists
}

type DB struct {
	mutex           sync.RWMutex
	tables          map[string]*BPTable
	currentDB       string                         // 当前的数据库
	currentTable    string                         // 当前的表
	databases       map[string]map[string]*BPTable // 存储每个数据库的表
	initFilePath    string
	operateFilePath string
}

// Use 切换当前使用的数据库
func (db *DB) Use(databaseName string) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// 检查数据库是否存在
	if _, exists := db.databases[databaseName]; !exists {
		fmt.Printf("数据库 %s 不存在\n", databaseName)
		return
	}
	// 切换当前使用的数据库
	db.currentDB = databaseName
	fmt.Printf("切换到数据库 %s\n", databaseName)
	filePath := filepath.Join(db.initFilePath, db.currentDB)
	db.operateFilePath = filePath
	ChangeWorkingDirectory(db.operateFilePath)
	fmt.Println("Use是否切换成功", db.operateFilePath)
}
func ChangeWorkingDirectory(path string) {
	err := os.Chdir(path)
	if err != nil {
		fmt.Printf("切换工作路径失败: %v\n", err)
	} else {
		fmt.Printf("成功切换到工作路径: %s\n", path)
	}
}

// ColumnType 表示列的数据类型
type ColumnType int

const (
	IntType ColumnType = iota
	StringType
)

// Column 定义了表中的一列
type Column struct {
	Name string     // 列的名称
	Type ColumnType // 列的数据类型
}

// TableSchema 定义了表的结构
type TableSchema struct {
	Columns []Column
}

// BPTable b+树中表的类型
type BPTable struct {
	Name   string
	Tree   *BPTree
	Schema TableSchema
}

func NewBPTable(name string, schema TableSchema) *BPTable {
	width := 4 // 这个值可以根据实际需要调整
	return &BPTable{
		Name:   name,
		Tree:   NewBPTree(width),
		Schema: schema,
	}
}

func NewDB() *DB {
	getwd, err := os.Getwd()
	if err != nil {
		fmt.Println("为获取到当前路径")
		return nil
	}
	return &DB{
		databases:    make(map[string]map[string]*BPTable),
		initFilePath: getwd,
	}
}

// CreateDatabase 创建数据库
func (db *DB) CreateDatabase(databaseName string) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// 检查数据库是否已经存在
	if _, exists := db.databases[databaseName]; exists {
		fmt.Printf("数据库 %s 已经存在\n", databaseName)
		return
	}

	db.databases[databaseName] = make(map[string]*BPTable)
	db.currentDB = databaseName

	fmt.Printf("数据库 %s 成功创建\n", databaseName)
	finalPath := path.Join(db.initFilePath, databaseName)
	db.operateFilePath = finalPath
	fmt.Println("创建数据库的路径", finalPath)
	err := os.Mkdir(finalPath, 0755)
	if err != nil {
		fmt.Println("创建数据库文件夹是否成功")
		return
	}

}

func (db *DB) CreateTable(tableName string, schema TableSchema) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// 检查是否选择了数据库
	if db.currentDB == "" {
		fmt.Println("没有选择数据库")
		return
	}

	// 检查表是否已经存在
	if _, exists := db.databases[db.currentDB][tableName]; exists {
		fmt.Printf("表 %s 已经存在\n", tableName)
		return
	}

	db.databases[db.currentDB][tableName] = NewBPTable(tableName, schema)
	fmt.Printf("表 %s 成功创建\n", tableName)

}

func (db *DB) Insert(tableName string, data map[string]interface{}) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	table, exists := db.databases[db.currentDB][tableName]
	if !exists {
		fmt.Printf("表 %s 不存在\n", tableName)
		return
	}
	keyColumn := table.Schema.Columns[0].Name
	key, ok := data[keyColumn].(int64)
	if !ok {
		fmt.Printf("Invalid key for table %s\n", tableName)
		return
	}
	table.Tree.Insert(key, data)

}

func (db *DB) Update(tableName string, data map[string]interface{}) bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	table, exists := db.databases[db.currentDB][tableName]
	if !exists {
		fmt.Printf("表 %s 不存在\n", tableName)
		return false
	}
	// 调用 BPTable 中的 Update 方法
	return table.Tree.Update(data["ID"].(int64), data)
}
func (t *BPTree) Update(key int64, value map[string]interface{}) bool {
	if _, exists := t.Select(key); exists {
		t.Set(key, value)
		return true
	}
	return false
}

func (db *DB) Select(tableName string, key int64) interface{} {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	table, exists := db.databases[db.currentDB][tableName]
	if !exists {
		fmt.Printf("表 %s 不存在t\n", tableName)
		return nil
	}
	return table.Tree.Get(key)
}

func (db *DB) Delete(tableName string, key int64) bool {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	table, exists := db.databases[db.currentDB][tableName]
	if !exists {
		fmt.Printf("Table %s does not exist\n", tableName)
		return false
	}
	table.Tree.Remove(key)
	_, exists = table.Tree.Select(key)
	return !exists
}

func ParseSQL(sql string, db *DB) bool {
	sql = strings.TrimSpace(sql)
	sql = strings.ToUpper(sql)
	sql = strings.TrimSuffix(sql, ";")
	words := strings.FieldsFunc(sql, func(r rune) bool {
		// 分割字符为逗号、左括号、右括号和空格
		return r == ',' || r == '(' || r == ')' || unicode.IsSpace(r)
	})

	fmt.Println(words)
	switch words[0] {
	case "EXIT":
		return false
	case "USE":
		db.Use(words[1])
	case "HELP":
		db.GetHelp()
	case "CREATE":
		if words[1] == "DATABASE" {
			db.CreateDatabase(words[2])
		} else if words[1] == "TABLE" {
			//[CREATE TABLE USER ID INT NAME STRING AGE INT ;]
			tableName := words[2]
			// 解析表结构
			columns := make([]Column, 0)
			for i := 3; i < len(words); i += 2 {
				// 检查是否有足够的单词
				if i+1 >= len(words) {
					fmt.Println("列定义缺少类型")
					return true
				}

				columnName := words[i]
				columnType := words[i+1]
				var colType ColumnType
				switch columnType {
				case "INT":
					colType = IntType
				case "STRING":
					colType = StringType
				default:
					fmt.Println("未知的字段类型:", columnType)
					return true
				}
				columns = append(columns, Column{Name: columnName, Type: colType})
			}

			// 定义表结构
			tableSchema := TableSchema{
				Columns: columns,
			}

			// 创建表
			db.CreateTable(tableName, tableSchema)
			//db.CreateTableFile(tableName, tableSchema)
		} else {
			fmt.Println("Invalid CREATE statement.")
		}
	case "INSERT":
		// [INSERT INTO USER ID NAME AGE VALUES 1 '阿亮' 22 ;]
		tableName := words[2]
		i := 3
		columns := make([]string, 0)
		for words[i] != "VALUES" {
			columns = append(columns, words[i])
			i++
		}
		i++ // 跳过 "VALUES"
		values := make([]interface{}, 0)
		for i < len(words) {
			var value interface{}
			if intValue, err := strconv.ParseInt(words[i], 10, 64); err == nil {
				value = intValue
			} else {
				value = words[i]
			}
			values = append(values, value)
			i++
		}
		if len(columns) != len(values) {
			fmt.Println("列的数量和值的数量不匹配")
			return true
		}
		data := make(map[string]interface{})
		for i, column := range columns {
			data[column] = values[i]
		}
		db.Insert(tableName, data)
		//updateData := db.SelectAll(tableName)
		//db.UpdateDataToFile(tableName, updateData)
		updateData := db.SelectAll(tableName)

		// 将map[int64]interface{}转换为map[string]interface{}
		convertedData := make(map[string]interface{})
		for key, value := range updateData {
			convertedData[strconv.FormatInt(key, 10)] = value
		}

		db.UpdateDataToFile(tableName, convertedData)
		//err := db.SaveDataToFile(tableName, data)
		//if err != nil {
		//	fmt.Println("insert插入文件失败")
		//	return
		//}
	case "UPDATE":
		// [UPDATE USER SET NAME = 1 AGE = 30 WHERE ID = 1;]
		tableName := words[1]
		i := 3
		data := make(map[string]interface{})
		for i < len(words) && words[i] != "WHERE" {
			// 列名
			columnName := words[i]
			i++

			// 判断是否是等号，如果不是，则语句无效
			if i >= len(words) || words[i] != "=" {
				fmt.Println("无效的更新表达式:", words[i-1])
				return true
			}
			i++

			// 值
			columnValue := words[i]
			i++

			// 存储到 data 中
			data[columnName] = columnValue
		}

		// 检查是否提前结束循环
		if i >= len(words) || words[i] != "WHERE" {
			fmt.Println("缺少 WHERE 子句")
			return true
		}

		i++ // 跳过 "WHERE"

		// 处理 WHERE 子句
		keyColumn := words[i]
		i++

		// 判断是否是等号，如果不是，则语句无效
		if i >= len(words) || words[i] != "=" {
			fmt.Println("无效的 WHERE 子句")
			return true
		}
		i++

		// 获取主键值
		key, err := strconv.ParseInt(words[i], 10, 64)
		if err != nil {
			fmt.Println("无效的主键值:", words[i])
			return true
		}

		// 存储到 data 中
		data[keyColumn] = key

		// 调用 Update 函数
		success := db.Update(tableName, data)
		if success {
			fmt.Println("更新成功")
		} else {
			fmt.Println("更新失败")
			return true
		}
		//updateData := db.SelectAll(tableName)
		//db.UpdateDataToFile(tableName, updateData)
		updateData := db.SelectAll(tableName)

		// 将map[int64]interface{}转换为map[string]interface{}
		convertedData := make(map[string]interface{})
		for key, value := range updateData {
			convertedData[strconv.FormatInt(key, 10)] = value
		}

		db.UpdateDataToFile(tableName, convertedData)

	case "SELECT":
		// [SELECT * FROM USER WHERE ID = 1;]
		tableName := words[3]
		// [SELECT * FROM USER WHERE ID = 1;]
		key, err := strconv.ParseInt(words[7], 10, 64)
		if err != nil {
			fmt.Println("无效的主键值:", words[7])
			return true
		}
		result := db.Select(tableName, key)
		if result != nil {
			// 判断是否是map，如果是就将其转变为map[string]interface{}
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				fmt.Println("无效的查询结果类型")
				return true
			}

			fmt.Print("结果: ")
			for key, value := range resultMap {
				fmt.Printf("%s: %v ", key, value)
			}
			fmt.Println()
		} else {
			fmt.Println("未查到")
		}
	case "DELETE":
		// [DELETE FROM USER WHERE ID = 2;]
		tableName := words[2]
		// 修正此处，将关键字改为正确的列名
		key, err := strconv.ParseInt(words[6], 10, 64)
		if err != nil {
			fmt.Println("无效的主键值:", words[6])
			return true
		}
		db.Delete(tableName, key)
		//updateData := db.SelectAll(tableName)
		//db.UpdateDataToFile(tableName, updateData)
		updateData := db.SelectAll(tableName)

		// 将map[int64]interface{}转换为map[string]interface{}
		convertedData := make(map[string]interface{})
		for key, value := range updateData {
			convertedData[strconv.FormatInt(key, 10)] = value
		}

		db.UpdateDataToFile(tableName, convertedData)
	default:
		fmt.Println("无效的语句")
	}
	return true
}

func (db *DB) SaveDataToFile(tableName string, data map[string]interface{}) error {
	fileName := tableName + ".csv"
	filePath := filepath.Join(db.operateFilePath, fileName)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("打开文件失败", err)
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// 按照排序后的键写入数据
	for _, key := range keys {
		value := data[key]
		rowData := []string{key, fmt.Sprintf("%v", value)}
		err := writer.Write(rowData)
		if err != nil {
			fmt.Println("写入数据失败", err)
			return err
		}
	}
	// 写入数据
	//for _, key := range data {
	//	//rowData := []string{key, fmt.Sprintf("%v", value)}
	//
	//	err := writer.Write(rowData)
	//	if err != nil {
	//		fmt.Println("写入数据失败", err)
	//		return err
	//	}
	//}

	writer.Flush() // 刷新缓冲区

	fmt.Println("数据写入文件成功")
	return nil
}

func (db *DB) UpdateDataToFile(tableName string, data map[string]interface{}) error {
	fileName := tableName + ".csv"
	filePath := filepath.Join(db.operateFilePath, fileName)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("打开文件失败", err)
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// 对键进行排序
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// 按照排序后的键写入数据
	for _, key := range keys {
		value := data[key]
		rowData := []string{key, fmt.Sprintf("%v", value)}
		err := writer.Write(rowData)
		if err != nil {
			fmt.Println("写入数据失败", err)
			return err
		}
	}

	writer.Flush() // 刷新缓冲区

	fmt.Println("数据写入文件成功")
	return nil
}

// 获取帮助信息

func (db *DB) GetHelp() {

	filePath := filepath.Join("/Users/zhangxueliang/GolandProjects/AliangSQL/tools/help.txt")

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("打开文件创建文件失败")
		return
	}
	defer file.Close()

	fi, err := os.Stat(filePath)
	if err != nil {
		fmt.Println("os.Stat获取文件失败")
		return
	}
	if fi.Size() == ' ' {
		data := []byte("创建数据库语法: create database xxx;        // create database blog;\n使用数据库语法: use xxx;                    // use blog;\n创建表语法: create table xx (字段  类型,字段  类型); // create table user (id int,name string);\n插入语法: insert into xx (字段 , 字段) values (值,值); // insert into user (id ,name) values (1,'阿亮');\n查询语法: select * from xxx where xx = xx;      // select * from user where id = 1;\n修改语法: update xx set 字段 = 值  where 字段 = 值; //update user set name = '亮亮' where id = 1;\n删除语法: delete from xx where 字段 = 值 ;     // delete from user where id =1")
		err := os.WriteFile(filePath, data, 0644)
		if err != nil {
			fmt.Println("os.WriteFile 写入文件出错")
			return
		}
	} else {
		readFile, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("ReadFile出错")
			return
		}
		fmt.Println(string(readFile))
	}
}
