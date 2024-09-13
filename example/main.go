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
