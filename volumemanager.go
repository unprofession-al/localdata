package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"gopkg.in/yaml.v2"
)

type VolumeManagerConfig struct {
	Name             string `json:"name" yaml:"name"`
	DatasourceType   string `json:"datasource_type" yaml:"datasource_type"`
	DatasourceConfig string `json:"datasource_config" yaml:"datasource_config"`
	Mountpoint       string `json:"mountpoint" yaml:"mountpoint"`
	Devicename       string `json:"devicename" yaml:"devicename"`
	Filesystem       string `json:"filesystem" yaml:"filesystem"`
}

func NewVolumeManagerConfigs(in []byte) ([]VolumeManagerConfig, error) {
	out := []VolumeManagerConfig{}
	err := yaml.Unmarshal(in, &out)
	return out, err
}

type VolumeManager struct {
	name        string
	source      Source
	instance    Instance
	volume      Volume
	attachement Attachement
	mount       Mount
}

func NewVolumeManager(vc VolumeManagerConfig) (VolumeManager, error) {
	// instance
	svc := ec2metadata.New(session.New())
	md, err := svc.GetInstanceIdentityDocument()
	if err != nil {
		return VolumeManager{}, err
	}
	i := Instance{
		id:               md.InstanceID,
		availabilityZone: md.AvailabilityZone,
		region:           md.Region,
	}

	// source
	s, err := NewSource(vc.DatasourceType, vc.DatasourceConfig)
	if err != nil {
		return VolumeManager{}, err
	}

	v := VolumeManager{
		name:   vc.Name,
		source: s,
		volume: Volume{},
		attachement: Attachement{
			devicename: vc.Devicename,
		},
		mount: Mount{
			mountpoint: vc.Mountpoint,
			filesystem: vc.Filesystem,
		},
		instance: i,
	}
	return v, nil
}

func (vm VolumeManager) Tags() map[string]string {
	return map[string]string{
		"Name":              fmt.Sprintf("%s-%s", vm.name, vm.instance.id),
		"VolumeManagerName": vm.name,
		"ForInstance":       vm.instance.id,
	}
}

func (vm VolumeManager) TagSpecification(resoucurceType string) []*ec2.TagSpecification {
	tags := []*ec2.Tag{}
	for k, v := range vm.Tags() {
		t := &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}
		tags = append(tags, t)
	}
	return []*ec2.TagSpecification{
		{
			ResourceType: aws.String(resoucurceType),
			Tags:         tags,
		},
	}
}

func (vm VolumeManager) TagFilter() []*ec2.Filter {
	filters := []*ec2.Filter{}
	for k, v := range vm.Tags() {
		f := &ec2.Filter{
			Name:   aws.String(fmt.Sprintf("tag:%s", k)),
			Values: []*string{aws.String(v)},
		}
		filters = append(filters, f)
	}
	return filters
}

func (vm VolumeManager) session() *session.Session {
	return session.Must(session.NewSession(&aws.Config{
		Region: aws.String(vm.instance.region)},
	))
}

func (vm VolumeManager) Ensure() error {
	session := vm.session()
	volumeId, volumeExists, err := vm.volume.Find(session, vm.TagFilter())
	if err != nil {
		return err
	}
	if !volumeExists {
		volumeId, err = vm.source.CreateVolume(vm.instance.availabilityZone, vm.TagSpecification("volume"))
		if err != nil {
			return err
		}
	}

	err = vm.attachement.Ensure(session, vm.instance.id, volumeId)
	if err != nil {
		return err
	}

	err = vm.mount.Mount(vm.attachement.devicename)
	return err
}

type Volume struct{}

func (v Volume) Find(session *session.Session, filters []*ec2.Filter) (volumeId string, exists bool, err error) {
	svc := ec2.New(session)
	input := &ec2.DescribeVolumesInput{
		Filters: filters,
	}
	result, err := svc.DescribeVolumes(input)
	if err != nil {
		return
	}
	if len(result.Volumes) == 0 {
		return
	}
	volumeId = *result.Volumes[0].VolumeId
	exists = true
	return
}

func (v Volume) Delete() error {
	return nil
}

type Attachement struct {
	devicename string
}

func (a Attachement) attached(session *session.Session, instanceId, volumeId string) (bool, error) {
	svc := ec2.New(session)
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeId),
		},
	}
	result, err := svc.DescribeVolumes(input)
	if err != nil {
		return false, err
	}
	if len(result.Volumes) == 0 {
		return false, fmt.Errorf("No volume '%s' found", volumeId)
	}

	volume := *result.Volumes[0]
	if len(volume.Attachments) == 0 {
		return false, nil
	} else if *volume.Attachments[0].InstanceId != instanceId {
		return true, fmt.Errorf("Volume '%s' attached to instance '%s' (should be '%s')", volumeId, *volume.Attachments[0].InstanceId, instanceId)
	}

	return true, nil
}

func (a Attachement) Ensure(session *session.Session, instanceId, volumeId string) error {
	svc := ec2.New(session)

	attached, err := a.attached(session, instanceId, volumeId)
	if err != nil {
		return err
	}

	if attached {
		return nil
	}

	input := &ec2.AttachVolumeInput{
		Device:     aws.String(a.devicename),
		InstanceId: aws.String(instanceId),
		VolumeId:   aws.String(volumeId),
	}
	_, err = svc.AttachVolume(input)
	if err != nil {
		return err
	}

	// TODO: properly wait for the attachement to be done
	time.Sleep(10 * time.Second)

	return nil
}

func (a Attachement) Detach() error {
	return nil
}

type Instance struct {
	id               string
	availabilityZone string
	region           string
}
