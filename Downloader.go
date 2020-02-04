package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const perm = 0755
const bucket = "j-artifacts"

var prBuilderRepo = ""
var prBuilderJobNumber = -1

func main() {
	// Parse given arguments
	flag.StringVar(&prBuilderRepo, "r", prBuilderRepo, "PR builder name like Hazelcast-EE-3.maintenance-sonar")
	flag.IntVar(&prBuilderJobNumber, "jn", prBuilderJobNumber, "PR builder job number")

	flag.Parse()

	if len(prBuilderRepo) < 1 {
		exit(fmt.Errorf("Enter a valid repo name\n"))
	}

	if prBuilderJobNumber < 1 {
		exit(fmt.Errorf("Enter a valid pr builder job number\n"))
	}

	compressedFileName := strconv.Itoa(prBuilderJobNumber) + ".tar"

	localDownloadDirectory := filepath.Join(userHomeDir(), bucket, prBuilderRepo, strconv.Itoa(prBuilderJobNumber))
	// create local dir if it is not there
	err := os.MkdirAll(localDownloadDirectory, perm)

	// download remote file to local file
start:
	fromRemoteFilePath := filepath.Join(prBuilderRepo, strconv.Itoa(prBuilderJobNumber), prBuilderRepo+"-"+compressedFileName)
	toLocalFilePath := filepath.Join(localDownloadDirectory, compressedFileName)
	err = download(fromRemoteFilePath, toLocalFilePath)
	if err != nil {
		aerr := err.(awserr.Error)
		if aerr.Code() == s3.ErrCodeNoSuchKey && strings.Contains(compressedFileName, ".tar") {
			compressedFileName = strconv.Itoa(prBuilderJobNumber) + ".zip"
			goto start
		} else {
			exit(err)
		}
	}

	if strings.Contains(compressedFileName, ".tar") {
		fmt.Println("Untarring...")
		untar(compressedFileName, localDownloadDirectory)
	} else if strings.Contains(compressedFileName, ".zip") {
		fmt.Println("Unzipping...")
		unzip(compressedFileName, localDownloadDirectory)
	}

	err = os.Remove(filepath.Join(localDownloadDirectory, strconv.Itoa(prBuilderJobNumber)+".tar"))
	if err != nil {
		exit(err)
	}

	err = os.Remove(filepath.Join(localDownloadDirectory, strconv.Itoa(prBuilderJobNumber)+".zip"))
	if err != nil {
		exit(err)
	}

	fmt.Println("Done. Uncompressed file is here --> " + localDownloadDirectory)
}

func untar(tarFileName string, localDownloadDirectory string) {
	tarFilePath := filepath.Join(localDownloadDirectory, tarFileName)

	file, err := os.Open(tarFilePath)

	if err != nil {
		exit(err)
	}

	defer file.Close()

	var fileReader io.ReadCloser = file

	fileReader, err = gzip.NewReader(file)
	if err != nil {
		exit(err)
	}

	tarBallReader := tar.NewReader(fileReader)

	for {
		header, err := tarBallReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			exit(err)
		}

		// get the individual filename and extract to the current directory
		filename := filepath.Join(localDownloadDirectory, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// handle directory
			err = os.MkdirAll(filename, 0755) // or use 0755 if you prefer
			if err != nil {
				exit(err)
			}

		case tar.TypeReg:
			// handle normal file
			writer, err := os.Create(filename)
			if err != nil {
				exit(err)
			}

			io.Copy(writer, tarBallReader)

			err = os.Chmod(filename, os.FileMode(header.Mode))

			if err != nil {
				exit(err)
			}

			writer.Close()
		default:
			fmt.Printf("Unable to untar type : %c in file %s", header.Typeflag, filename)
		}
	}
}

func download(fromRemoteFilePath string, toLocalFilePath string) error {
	// Connect to s3
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		return err
	}

	file, err := os.Create(toLocalFilePath)

	downloader := s3manager.NewDownloader(sess)
	fmt.Println("Download started...")
	_, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fromRemoteFilePath),
	})

	return err
}

func exit(err error) {
	fmt.Println(err)
	os.Exit(-1)
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	} else if runtime.GOOS == "linux" {
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
	}
	return os.Getenv("HOME")
}
