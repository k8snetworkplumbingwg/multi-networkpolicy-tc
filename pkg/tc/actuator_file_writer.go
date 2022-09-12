package tc

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	"github.com/Mellanox/multi-networkpolicy-tc/pkg/utils"
)

// NewActuatorFileWriterImpl returns a new ActuatorFileWriterImpl instance
func NewActuatorFileWriterImpl(path string, log klog.Logger) *ActuatorFileWriterImpl {
	return &ActuatorFileWriterImpl{
		log:  log,
		path: path,
	}
}

// ActuatorFileWriterImpl implements Actuator interface and is used to save TC objects to file
type ActuatorFileWriterImpl struct {
	log  klog.Logger
	path string
}

// Actuate implements Actuator interface
// Note(adrianc): As we are saving tc objects (mainly filters) to file
// in a human-readable format (as this is really intended for debug purposes). We need represent
// these objects as string. For now, we leverage CmdLineGenerator interface which is implemented by all objects.
// Later on, it may be desired to extend the interface with String() method and implement throughout then use it here.
func (a ActuatorFileWriterImpl) Actuate(objects *Objects) error {
	exist, err := utils.PathExists(a.path)
	if err != nil {
		return errors.Wrapf(err, "failed to determine if path exist: %s", a.path)
	}

	currentBuf := bytes.NewBuffer([]byte{})
	if exist {
		data, err := os.ReadFile(a.path)
		if err != nil {
			klog.Warningf("failed to read file at: %s. error: %s", a.path, err)
		} else {
			currentBuf = bytes.NewBuffer(data)
		}
	}

	newBuf := bytes.Buffer{}
	if objects.QDisc == nil {
		_, _ = newBuf.WriteString("qdisc: <nil>\n")
	} else {
		_, _ = newBuf.WriteString(fmt.Sprintf("qdisc: %s\n",
			strings.Join(objects.QDisc.GenCmdLineArgs(), " ")))
	}

	_, _ = newBuf.WriteString("filters:\n")
	for _, f := range objects.Filters {
		_, _ = newBuf.WriteString(strings.Join(f.GenCmdLineArgs(), " "))
		_, _ = newBuf.WriteRune('\n')
	}

	if bytes.Equal(currentBuf.Bytes(), newBuf.Bytes()) {
		klog.Info("current and new rules are the same - no action needed.")
		return nil
	}

	klog.Infof("saving new rules to: %s", a.path)

	file, err := os.Create(a.path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = newBuf.WriteTo(file)
	return err
}
