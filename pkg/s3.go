// Package pkg
/*****
@author    Jonas Kaninda
@license   MIT License <https://opensource.org/licenses/MIT>
@Copyright © 2024 Jonas Kaninda
**/
package pkg

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jkaninda/mysql-bkup/utils"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// CreateSession creates a new AWS session
func CreateSession() (*session.Session, error) {
	awsConfig := initAWSConfig()
	// Configure to use MinIO Server
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(awsConfig.accessKey, awsConfig.secretKey, ""),
		Endpoint:         aws.String(awsConfig.endpoint),
		Region:           aws.String(awsConfig.region),
		DisableSSL:       aws.Bool(awsConfig.disableSsl),
		S3ForcePathStyle: aws.Bool(awsConfig.forcePathStyle),
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
	utils.Info("Download data from S3 storage...")
	file, err := os.Create(filepath.Join(destinationPath, key))
	if err != nil {
		utils.Error("Failed to create file", err)
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
		utils.Error("Failed to download file %s", key)
		return err
	}
	utils.Info("Backup downloaded:  %s bytes size %s ", file.Name(), numBytes)

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
					utils.Info("Failed to delete object %s: %v", *object.Key, err)
				} else {
					utils.Info("Deleted object %s\n", *object.Key)
				}
			}
		}
		return !lastPage
	})
	if err != nil {
		utils.Error("Failed to list objects: %v", err)
	}

	utils.Info("Finished deleting old files.")
	return nil
}
