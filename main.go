package main

import (
	"batchRequestsRecover/cmd"
	"fmt"
)

func main() {
	fmt.Println("Starting batch requests recover")
	cmd.Run()
	fmt.Println("Batch requests recover Ended")

}
