# FINS

基于golang实现的欧姆龙fins协议组件库，内部实现TCP、UDP连接

## 例



```golang
package main

import (
	"fmt"
	"github.com/zwzszwzs/fins"
)

func main() {
	c, err := fins.NewClient("udp", "10.2.30.71", "5010", "", "", 0, 10, 0, 0, 1, 0)
	// c, err := fins.NewClient("tcp", "10.2.30.71", "1025", "", "", 0, 10 ,0, 0 ,1, 0)
	if err != nil {
		panic(err)
	}
	fmt.Println("connected successed")
	defer c.Close()

	// cio
	dataCIOBits, err := c.ReadBits(fins.MemoryAreaCIOBit, 100, 0, 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(dataCIOBits)

	// DM
	dataCIOWord, err := c.ReadWordsToUint16(fins.MemoryAreaDMWord, 100, 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(dataCIOWord)

	// HR
	dataHRBits, err := c.ReadBits(fins.MemoryAreaHRWord, 100, 0, 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(dataHRBits)
}
```



## 读取接口

```
ReadWordsToUint16(byte, uint16, uint16) ([]uint16, error)
ReadWordsToUint32(byte, uint16, uint16) ([]uint32, error)
ReadBytes(byte, uint16, uint16) ([]byte, error)
ReadString(byte, uint16, uint16) (string, error)
ReadBits(byte, uint16, byte, uint16) ([]bool, error)
```



## 写入接口

```
WriteWords(byte, uint16, []uint16) error
WriteString(byte, uint16, string) error
WriteBytes(byte, uint16，[]byte) error
WriteBits(byte, uint16, byte, []bool) error
```



## 特性

- 接口灵活多变
- 读取出连续地址的数值
- 读取的数据类型可自定义-uint16、uint32、string、bool、byte
- 支持TCP、UDP连接




