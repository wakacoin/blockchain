package wakacoin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"time"
)

func CliCheckErr(err error) {
	if err != nil {
		fmt.Println("\n", err)
		os.Exit(1)
	}
}

func CheckErr(err error) {
	if err != nil {
		t := time.Now().UTC()
		timeNow := t.Format("02 Jan 15:04:05")
		fmt.Println(timeNow)
		
		// msg := blockChain + " blockchain, panic."
		
		msg := fmt.Sprintf("%s blockchain, panic: %s", blockChain, err)
		
		SendAdminEmail(msg)
		
		panic(err)
	}
}

func PrintErr(err interface{}) {
	if err != nil {
		t := time.Now().UTC()
		timeNow := t.Format("02 Jan 15:04:05")
		fmt.Println(timeNow, err)
	}
}

func PrintMessage(str string) {
	if len(str) != 0 {
		t := time.Now().UTC()
		timeNow := t.Format("02 Jan 15:04:05")
		fmt.Println(timeNow, str)
	}
}

func Uint8ToByte(num uint8) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	CheckErr(err)

	return buf.Bytes()
}

func Uint32ToByte(num uint32) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	CheckErr(err)

	return buf.Bytes()
}

func Int64ToByte(num int64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	CheckErr(err)

	return buf.Bytes()
}

func Uint64ToByte(num uint64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	CheckErr(err)

	return buf.Bytes()
}

func ByteToUint32(b []byte) (num uint32) {
	bytesBuffer := bytes.NewBuffer(b)
	
	binary.Read(bytesBuffer, binary.BigEndian, &num)
	
	return num
}

func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

func ReverseHashes(data [][32]byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

func Reverse20ByteHashes(data [][20]byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

func generateRandomNumber(start int, end int, count int) []int {
	if end < start || (end-start) < count {
		return nil
	}
	
	nums := make([]int, 0)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	for len(nums) < count {
		num := r.Intn((end - start)) + start
		
		exist := false
		
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}
		
		if !exist {
			nums = append(nums, num)
		}
	}
	
	return nums
}