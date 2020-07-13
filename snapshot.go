package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const snapshotsourcetype = "snapshot"

func init() {
	RegisterSourceType(snapshotsourcetype, NewSnapshotSource)
}

type Snapshot struct {
	id     string
	region string
}

func NewSnapshotSource(config string) (Source, error) {
	parts := strings.Split(config, ":")
	if len(parts) != 2 {
		return Snapshot{}, fmt.Errorf("Source config for Snapshot fucked up, is '%s', must be '[id]:[region]'", config)
	}
	return Snapshot{
		id:     parts[0],
		region: parts[1],
	}, nil
}

func (s Snapshot) session() *session.Session {
	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(s.region)},
	))
}

func (s Snapshot) available() (bool, error) {
	svc := ec2.New(s.session())
	input := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{
			aws.String(s.id),
		},
	}
	result, err := svc.DescribeSnapshots(input)
	if err != nil {
		return false, err
	}
	if len(result.Snapshots) == 0 {
		return false, nil
	}
	return true, nil
}

func (s Snapshot) CreateVolume(az string, tags []*ec2.TagSpecification) (string, error) {
	exists, err := s.available()
	if err != nil {
		fmt.Println("error while checking in snapshot is available", s)
		return "", err
	} else if !exists {
		return "", fmt.Errorf("Snapshot '%s' does not exist", s.id)
	}
	svc := ec2.New(s.session())
	input := &ec2.CreateVolumeInput{
		AvailabilityZone:  aws.String(az),
		VolumeType:        aws.String("gp2"),
		TagSpecifications: tags,
		SnapshotId:        &s.id,
	}

	result, err := svc.CreateVolume(input)
	if err != nil {
		return "", err
	}

	// TODO: properly wait for the volume to be created
	time.Sleep(10 * time.Second)

	return *result.VolumeId, nil
}

func (s Snapshot) Sync(chan SyncStatus) (bool, error) {
	return true, nil
}

func (s Snapshot) Synced() (bool, error) {
	return true, nil
}
