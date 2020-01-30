package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"
	"strconv"
)

const bucket = "j-artifacts"

var prBuilderType = "os"
var prBuilderJobNumber = 5654

func main() {
	// Parse given arguments
	flag.StringVar(&prBuilderType, "t", prBuilderType, "PR builder type: os or ee")
	flag.IntVar(&prBuilderJobNumber, "j", prBuilderJobNumber, "PR builder job number")

	flag.Parse()

	if prBuilderJobNumber < 0 {
		exit(fmt.Errorf("Enter a valid pr builder job number\n"))
	}

	pathToFile := checkAndGetFullFilePath()
	localFileName := strconv.Itoa(prBuilderJobNumber) + ".zip"

	err := download(pathToFile, localFileName)

	fmt.Println("Unzipping...")
	err = doUnzip(localFileName, bucket+"/"+prBuilderType+"/"+strconv.Itoa(prBuilderJobNumber))

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("All done.")
}

func download(pathToFile string, localFileName string) error {
	// Connect to s3
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		panic(err)
	}

	file, err := os.Create(localFileName)

	downloader := s3manager.NewDownloader(sess)
	fmt.Println("Download started...")
	_, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(pathToFile),
	})

	if err != nil {
		fmt.Println(err)
		return nil
	}
	return err
}

func checkAndGetFullFilePath() (path string) {
	jobNumberStr := strconv.Itoa(prBuilderJobNumber)

	if prBuilderType == "os" {
		return "Hazelcast-pr-builder/" + jobNumberStr + "/Hazelcast-pr-builder-" + jobNumberStr + ".zip"
	}

	if prBuilderType == "ee" {
		return "Hazelcast-EE-pr-builder/" + jobNumberStr + "/Hazelcast-EE-pr-builder-" + jobNumberStr + ".zip"
	}

	exit(fmt.Errorf("Unsupported prBuilderType name: " + prBuilderType))
	return ""
}

func exit(err error) {
	fmt.Println(err)
	os.Exit(-1)
}
