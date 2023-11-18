## AliangSQL  利用go 开发基于B+树的小型关系型数据库 
开发环境:mac
1.终端输入：git clone https://github.com/aaaaaaliang/AliangSQL.git
2.main 方法里直接运行即可

## 目的
“What I cannot create, I do not understand.” – Richard Feynman

正如这句名言，理解一个事物最好的办法就是亲自设计制作它，本文将介绍一个简单的关系型数据库系统（类似MySQL、sqlite等）的开发过程，用于理解关系型数据库基本工作原理，我称它为AliangSQL。

## 帮助界面
![image](https://github.com/aaaaaaliang/AliangSQL/assets/117182742/f9c6411d-7ab8-40bb-aa96-3d3d11598f79)

1.使用go开发；
2.已实现基本的CURD操作，使用控制台SQL：
  <ol type="i">
  <li>插入语法: `insert into xx (字段 , 字段) values (值,值);` // insert into user (id ,name) values (1,'阿亮');</li>
  <li>查询语法: `select * from xxx where xx = xx;`      // select from user where id = 1;</li>
  <li>修改语法: `update xx set 字段 = 值  where 字段 = 值;` //update user set name = '亮亮' where id = 1;</li>
  <li>删除语法: `delete from xx where 字段 = 值 ;`     // delete from user where id =1</li>
</ol>


3.底层使用B+树（B+ TREE）构建索引；
4.利用.csv文件存储表

## 过程
![image](https://github.com/aaaaaaliang/AliangSQL/assets/117182742/9026eb1a-3820-4a09-b91e-1324fd48f574)

主要开发步骤
<ol>
  <li>创建一个控制台对话交互程序（REPL：read-execute-print loop）；</li>
  <li>创建一个简单的词法分析器用来解析SQL语句；</li>
  <li>编写CURD函数实现数据库的增删改查操作；</li>
  <li>创建一个B+树索引引擎，进行数据库的索引和磁盘读写操作，数据表将以二进制的形式存储。</li>
</ol>

b+树实现

### b+树特性
<ol>
  <li>每个节点至多有M个子树。</li>
  <li>除根结点外，每个结点至少有ceil(M/2)个子树。</li>
  <li>结点的子树个数于关键字个数相等。</li>
  <li>所有的叶子结点中包含了全部关键字的信息，及指向含这些关键字记录的指针，且叶子结点本身依关键字的大小自小而大顺序链接。</li>
  <li>所有的非终端结点（非叶子结点）可以看成是索引部分，结点中仅含有其子树（根结点）中的最大（或最小）关键字。</li>
</ol>

![image](https://github.com/aaaaaaliang/AliangSQL/assets/117182742/b80e8362-d35d-456c-b0c9-f6442695a57b)



