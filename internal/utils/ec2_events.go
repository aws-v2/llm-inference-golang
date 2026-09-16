package utils

import (
	"encoding/json"
	"fmt"
	"llm-inference-service/internal/nats"
	"log"
)


func  ReleaseSageMakerVM(nc *nats.Client, vmId string) error {

	subject := "ec2.task.release_lambda"
	type ReleaseVMReq struct {
		VMID string `json:"vm_id"`
	}

	req := ReleaseVMReq{
		VMID: vmId,
	}

	log.Printf("Thisis ht ereq being sent ;  %+v", req)

	respBytes, err := nc.Publisher.Request(subject, req)

	if err != nil {
		fmt.Printf("\nError sending reelease vmevent  %v", err)
		return fmt.Errorf("Error sending reelease vmevent  %v", err)
	}

	type RealeaseResponse struct {
		Code int `json:"code"`
	}

	var releaseVmresp RealeaseResponse
	err = json.Unmarshal(respBytes, &releaseVmresp)

	if err != nil {
		fmt.Printf("\nError unmarshalling  reelease vmevent response   %v", err)
		return fmt.Errorf("Error unmarshalling  reelease vmevent response  %v", err)

	}
	if releaseVmresp.Code != 200 {
		fmt.Printf("\nError in EC@ Couldnot release the vm:    %v", releaseVmresp.Code)
		return fmt.Errorf("Error in EC@ Couldnot release the vm:    %v", releaseVmresp.Code)

	} else {
		fmt.Printf("\n EC2  released the vm succesfully:    %v", releaseVmresp.Code)
	}

	return nil

}