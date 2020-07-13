package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

type Mount struct {
	mountpoint string
	filesystem string
}

func (m Mount) createMountpoint() error {
	src, err := os.Stat(m.mountpoint)

	if os.IsNotExist(err) {
		err = os.MkdirAll(m.mountpoint, 0755)
		if err != nil {
			return err
		}
		return nil
	}

	if src.Mode().IsRegular() {
		return fmt.Errorf("Mountpoint '%s' already exists as a file", m.mountpoint)
	}

	return err
}

func (m Mount) mounted(devicename string) (bool, error) {
	file, err := os.Open(mountinfo)
	if err != nil {
		return false, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	info, err := ParseMountInfo(reader)

	for _, mi := range info {
		if mi.Root == m.mountpoint && mi.MountSource == devicename {
			return true, nil
		} else if mi.Root == m.mountpoint && mi.MountSource != devicename {
			return true, fmt.Errorf("Mountpoint '%s' is already used by different device '%s'", m.mountpoint, devicename)
		} else if mi.Root != m.mountpoint && mi.MountSource == devicename {
			return true, fmt.Errorf("Device '%s' is already mounted at '%s'", devicename, m.mountpoint)
		}
	}

	return false, nil
}

func (m Mount) Mount(devicename string) error {
	mounted, err := m.mounted(devicename)
	if err != nil {
		return err
	}

	if mounted {
		return nil
	}

	err = m.createMountpoint()
	if err != nil {
		return err
	}
	return syscall.Mount(devicename, m.mountpoint, m.filesystem, 0, "")
}

func (m Mount) Unmount(devicename string) error {
	mounted, err := m.mounted(devicename)
	if err != nil {
		return err
	}

	if !mounted {
		return nil
	}

	return syscall.Unmount(m.mountpoint, 0)
}

// mountinfo https://github.com/fntlnz/mountinfo/blob/master/mountinfo.go
const mountinfo = "/proc/self/mountinfo"

type Mountinfo struct {
	MountID        string
	ParentID       string
	MajorMinor     string
	Root           string
	MountPoint     string
	MountOptions   string
	OptionalFields string
	FilesystemType string
	MountSource    string
	SuperOptions   string
}

func getMountPart(pieces []string, index int) string {
	if len(pieces) > index {
		return pieces[index]
	}
	return ""
}

// GetMountInfo opens a mountinfo file, returns
func GetMountInfo(fd string) ([]Mountinfo, error) {
	file, err := os.Open(fd)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	return ParseMountInfo(file)
}

// ParseMountInfoString transforms a mountinfo string in a struct of type Mountinfo
func ParseMountInfoString(tx string) *Mountinfo {
	pieces := strings.Split(tx, " ")
	count := len(pieces)
	if count < 1 {
		return nil
	}
	i := strings.Index(tx, " - ")
	postFields := strings.Fields(tx[i+3:])
	preFields := strings.Fields(tx[:i])
	return &Mountinfo{
		MountID:        getMountPart(preFields, 0),
		ParentID:       getMountPart(preFields, 1),
		MajorMinor:     getMountPart(preFields, 2),
		Root:           getMountPart(preFields, 3),
		MountPoint:     getMountPart(preFields, 4),
		MountOptions:   getMountPart(preFields, 5),
		OptionalFields: getMountPart(preFields, 6),
		FilesystemType: getMountPart(postFields, 0),
		MountSource:    getMountPart(postFields, 1),
		SuperOptions:   getMountPart(postFields, 2),
	}
}

// ParseMountInfo parses the mountinfo content from an io.Reader, e.g a file
func ParseMountInfo(buffer io.Reader) ([]Mountinfo, error) {
	info := []Mountinfo{}
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		tx := scanner.Text()
		info = append(info, *ParseMountInfoString(tx))
	}

	if err := scanner.Err(); err != nil {
		return info, err
	}
	return info, nil
}
