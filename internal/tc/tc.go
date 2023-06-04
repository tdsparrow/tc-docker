package tc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CodyGuo/glog"
	"github.com/CodyGuo/tc-docker/internal/docker"
	"github.com/CodyGuo/tc-docker/pkg/command"
)

var (
	ErrTcNotFound = errors.New("RTNETLINK answers: No such file or directory")
)

func setTcRate(device, rate, ceil, netns string) error {
	// check netns, if empty, compose ip netns exec command prefix
	if netns != "" {
		netns = fmt.Sprintf("ip netns exec %s ", netns)
	}
	delRootHandleCmd := fmt.Sprintf("%s/usr/sbin/tc qdisc del dev %s root", netns, device)
	glog.Debug(delRootHandleCmd)
	out, err := command.CombinedOutput(delRootHandleCmd)
	if err != nil {
		if strings.TrimSpace(string(out)) != ErrTcNotFound.Error() {
			return fmt.Errorf("out: %s, error: %v", out, err)
		}
	}
	addRootHandleCmd := fmt.Sprintf("%s/usr/sbin/tc qdisc add dev %s root handle 1a1a: htb default 1", netns, device)
	glog.Debug(addRootHandleCmd)
	out, err = command.CombinedOutput(addRootHandleCmd)
	if err != nil {
		return fmt.Errorf("out: %s, error: %v", out, err)
	}
	addClassCmd := fmt.Sprintf("%s/usr/sbin/tc class add dev %s parent 1a1a: classid 1a1a:1 htb rate %s ceil %s", netns, device, rate, ceil)
	glog.Debug(addClassCmd)
	out, err = command.CombinedOutput(addClassCmd)
	if err != nil {
		return fmt.Errorf("out: %s, error: %v", out, err)
	}
	addSfqHandleCmd := fmt.Sprintf("%s/usr/sbin/tc qdisc add dev %s parent 1a1a:1 handle 2a2a: sfq perturb 10", netns, device)
	glog.Debug(addSfqHandleCmd)
	out, err = command.CombinedOutput(addSfqHandleCmd)
	if err != nil {
		return fmt.Errorf("out: %s, error: %v", out, err)
	}
	addFilterCmd := fmt.Sprintf("%s/usr/sbin/tc filter add dev %s parent 1a1a: protocol ip prio 1 u32 match ip src 0.0.0.0/0 match ip dst 0.0.0.0/0 flowid 1a1a:1", netns, device)
	glog.Debug(addFilterCmd)
	out, err = command.CombinedOutput(addFilterCmd)
	if err != nil {
		return fmt.Errorf("out: %s, error: %v", out, err)
	}
	return nil
}

func setTcRateHost(device, rate, ceil string) error {
	return setTcRate(device, rate, ceil, "")
}

func setTcRateContainer(device, rate, ceil, containerName string) error {
	return setTcRate(device, rate, ceil, containerName)
}

func SetTcRate(container *docker.Container) error {
	err := setTcRateHost(container.VethPeer, container.TcRate, container.TcCeil)
	if err != nil {
		return err
	}
	return setTcRateContainer(container.Veth, container.TcRate, container.TcCeil, container.Name)
}
