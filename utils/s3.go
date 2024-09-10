// Package utils /
/*****
@author    Jonas Kaninda
@license   MIT License <https://opensource.org/licenses/MIT>
@Copyright © 2024 Jonas Kaninda
**/
package utils

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// CreateSession creates a new AWS session
func CreateSession() (*session.Session, error) {
	// AwsVars Required environment variables for AWS S3 storage
	var awsVars = []string{
		"AWS_S3_ENDPOINT",
		"AWS_S3_BUCKET_NAME",
		"AWS_ACCESS_KEY",
		"AWS_SECRET_KEY",
		"AWS_REGION",
		"AWS_REGION",
		"AWS_REGION",
	}

	endPoint := GetEnvVariable("AWS_S3_ENDPOINT", "S3_ENDPOINT")
	accessKey := GetEnvVariable("AWS_ACCESS_KEY", "ACCESS_KEY")
	secretKey := GetEnvVariable("AWS_SECRET_KEY", "SECRET_KEY")
	_ = GetEnvVariable("AWS_S3_BUCKET_NAME", "BUCKET_NAME")

	region := os.Getenv("AWS_REGION")
	awsDisableSsl, err := strconv.ParseBool(os.Getenv("AWS_DISABLE_SSL"))
	if err != nil {
		Fatal("Unable to parse AWS_DISABLE_SSL env var: %s", err)
	}

	err = CheckEnvVars(awsVars)
	if err != nil {
		Fatal("Error checking environment variables\n: %s", err)
	}
	// S3 Config
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:         aws.String(endPoint),
		Region:           aws.String(region),
		DisableSSL:       aws.Bool(awsDisableSsl),
		S3ForcePathStyle: aws.Bool(true),
	}
	return session.NewSession(s3Config)

}

// UploadFileToS3 uploads a file to S3 with a given prefix
func UploadFileToS3(filePath, key, bucket, prefix string) error {
	sess, err := CreateSession()
	if err != nil {
		return err
	}

	svc := s3.New(sess)

	file, err := os.Open(filepath.Join(filePath, key))
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	objectKey := filepath.Join(prefix, key)

	buffer := make([]byte, fileInfo.Size())
	file.Read(buffer)
	fileBytes := bytes.NewReader(buffer)
	fileType := http.DetectContentType(buffer)

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(objectKey),
		Body:          fileBytes,
		ContentLength: aws.Int64(fileInfo.Size()),
		ContentType:   aws.String(fileType),
	})
	if err != nil {
		return err
	}

	return nil
}
func DownloadFile(destinationPath, key, bucket, prefix string) error {

	sess, err := CreateSession()
	if err != nil {
		return err
	}
	Info("Download backup from S3 storage...")
	file, err := os.Create(filepath.Join(destinationPath, key))
	if err != nil {
		fmt.Println("Failed to create file", err)
		return err
	}
	defer file.Close()

	objectKey := filepath.Join(prefix, key)

	downloader := s3manager.NewDownloader(sess)
	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectKey),
		})
	if err != nil {
		fmt.Println("Failed to download file", err)
		return err
	}
	Info("Backup downloaded:  %s bytes size %s ", file.Name(), numBytes)

	return nil
}
func DeleteOldBackup(bucket, prefix string, retention int) error {
	sess, err := CreateSession()
	if err != nil {
		return err
	}

	svc := s3.New(sess)

	// Get the current time and the time threshold for 7 days ago
	now := time.Now()
	backupRetentionDays := now.AddDate(0, 0, -retention)

	// List objects in the bucket
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	err = svc.ListObjectsV2Pages(listObjectsInput, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, object := range page.Contents {
			if object.LastModified.Before(backupRetentionDays) {
				// Object is older than retention days, delete it
				_, err := svc.DeleteObject(&s3.DeleteObjectInput{
					Bucket: aws.String(bucket),
					Key:    object.Key,
				})
				if err != nil {
					log.Printf("Failed to delete object %s: %v", *object.Key, err)
				} else {
					fmt.Printf("Deleted object %s\n", *object.Key)
				}
			}
		}
		return !lastPage
	})
	if err != nil {
		log.Fatalf("Failed to list objects: %v", err)
	}

	fmt.Println("Finished deleting old files.")
	return nil
}
