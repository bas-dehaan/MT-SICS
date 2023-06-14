package MT_SICS

import (
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"io"
	"math"
	"regexp"
	"strconv"
	"time"
)

// Connect to the scale via the given port
//
// Inputs:
//   - port: the port to connect to, e.g. COM1
//
// Outputs:
//   - io.ReadWriteCloser: the connection to the scale
//   - error
func Connect(port string) (io.ReadWriteCloser, error) {
	options := serial.OpenOptions{
		PortName:        port,
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	return serial.Open(options)
}

// DirectCommand sends a command to the scale and waits for the response of the MT-balance.
// The response is tested against the given regex, which should match the response within the timeout of 5 seconds.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//   - command: the command to send to the scale
//   - regex: the regular expression to match the response from the scale
//
// Outputs:
//   - []byte: the response from the scale
//   - error: most likely a timeout error, caused by the regex not matching to the response within 5 seconds
func DirectCommand(connection io.ReadWriteCloser, command string, regex *regexp.Regexp) ([]byte, error) {
	// Write
	_, err := connection.Write([]byte(command + "\r\n"))
	if err != nil {
		return nil, err
	}

	// Read (until match or timeout)
	buf := make([]byte, 128)
	match := false
	start := time.Now()
	timeout := 5 * time.Second

	for !match && time.Since(start) < timeout {
		_, err = connection.Read(buf)
		if err != nil {
			return nil, err
		}

		match = regex.Match(buf)
	}

	if !match {
		err = fmt.Errorf("command '%s' timed-out, want: %s, got: %s", command, regex.String(), string(buf))
	}

	return buf, err
}

// SetTarget sets a target weight and tolerance on the scale.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//   - target: the target weight
//   - unit: the unit of the target weight, e.g. g
//   - upperTolerance: the upper tolerance of the target weight
//   - lowerTolerance: the lower tolerance of the target weight
//   - relativeTolerance: true if the tolerance is relative (in %) or false if absolute (in unit)
//
// Outputs:
//   - error: see DirectCommand()
func SetTarget(connection io.ReadWriteCloser, target float64, unit string, upperTolerance float64, lowerTolerance float64, relativeTolerance bool) error {
	regex := regexp.MustCompile(`A10 A`)

	targetString := "A10 0 " + strconv.FormatFloat(target, 'f', 2, 64) + " " + unit + ""
	_, err := DirectCommand(connection, targetString, regex)
	if err != nil {
		return err
	}

	if relativeTolerance {
		unit = "%"
	}

	upperToleranceString := "A10 1 " + strconv.FormatFloat(upperTolerance, 'f', 2, 64) + " " + unit + ""
	_, err = DirectCommand(connection, upperToleranceString, regex)
	if err != nil {
		return err
	}

	lowerToleranceString := "A10 2 " + strconv.FormatFloat(lowerTolerance, 'f', 2, 64) + " " + unit + ""
	_, err = DirectCommand(connection, lowerToleranceString, regex)
	return err
}

// SetResultID sets the result ID on the scale.
// The result ID is used to identify the measurement, e.g. the sample number or patient ID.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//   - label: the label of the result ID, e.g. "Sample No.:" or "Patient ID:"
//   - value: the value of the result ID, e.g. "1234" or "John Doe"
//
// Outputs:
//   - error: see DirectCommand()
func SetResultID(connection io.ReadWriteCloser, label string, value string) error {
	msgString := "A36 1 \"" + label + "\" \"" + value + "\""
	regex := regexp.MustCompile(`A36 A`)

	_, err := DirectCommand(connection, msgString, regex)
	return err
}

// SetTaskID sets the task ID on the scale.
// The task ID is used to identify the measurement step, e.g. a duplicate number or process step.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//   - label: the label of the task ID, e.g. "Duplicate No.:" or "Process step:"
//   - value: the value of the task ID, e.g. "1 of 2" or "1st weighing"
//
// Outputs:
//   - error: see DirectCommand()
func SetTaskID(connection io.ReadWriteCloser, label string, value string) error {
	msgString := "A37 1 \"" + label + "\" \"" + value + "\""
	regex := regexp.MustCompile(`A37 A`)

	_, err := DirectCommand(connection, msgString, regex)
	return err
}

// SetMessage sets a message on the display of the scale, overlaying the weight value.
// The character limit is dependent on the scale model.
// An empty string will clear the message.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//   - message: the message to display, e.g. "See PC for instructions"
//
// Outputs:
//   - error: see DirectCommand()
func SetMessage(connection io.ReadWriteCloser, message string) error {
	msgString := "D \"" + message + "\""
	regex := regexp.MustCompile(`D A`)

	_, err := DirectCommand(connection, msgString, regex)
	return err
}

// ShowWeight clears the message on the display of the scale, showing the weight value.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - error: see DirectCommand()
func ShowWeight(connection io.ReadWriteCloser) error {
	regex := regexp.MustCompile(`DW A`)
	_, err := DirectCommand(connection, "DW", regex)
	return err
}

// GetUnit retrieves the unit currently used by the scale.
// There are 3 channels on the scale:
//   - 0: Host unit, used on the MT-SICS connection
//   - 1: Display unit, used on the scale display
//   - 2: Info unit, used on the info field on the scale's display
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//   - channel: the channel to retrieve the unit from
//
// Outputs:
//   - unit: the unit used on the specified channel, e.g. "g"
//   - error: see DirectCommand()
func GetUnit(connection io.ReadWriteCloser, channel int) (string, error) {
	regex := regexp.MustCompile(`M21 A [0-9] ([a-zA-Z]+)`)
	buf, err := DirectCommand(connection, "M21 "+strconv.Itoa(channel), regex)
	if err != nil {
		return "", err
	}

	result := regex.FindStringSubmatch(string(buf))
	return result[1], nil
}

// SetUnit sets the unit used by the scale.
// There are 3 channels on the scale:
//   - 0: Host unit, used on the MT-SICS connection
//   - 1: Display unit, used on the scale display
//   - 2: Info unit, used on the info field on the scale's display
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//   - unit: the unit to use on the specified channel, e.g. "g". Make sure to use proper capitalization.
func SetUnit(connection io.ReadWriteCloser, unit string, channel int) error {
	regex := regexp.MustCompile(`M21 A`)
	_, err := DirectCommand(connection, "M21 "+strconv.Itoa(channel)+" "+unit, regex)
	return err
}

// PowerOn turns the scale on from stand-by mode.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - error: see DirectCommand()
func PowerOn(connection io.ReadWriteCloser) error {
	regex := regexp.MustCompile(`PWR [AL]`) // PWR L will be returned if the scale is already on
	_, err := DirectCommand(connection, "PWR 1", regex)
	return err
}

// PowerOff turns the scale into stand-by mode.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - error: see DirectCommand()
func PowerOff(connection io.ReadWriteCloser) error {
	regex := regexp.MustCompile(`PWR [AL]`) // PWR L will be returned if the scale is already off
	_, err := DirectCommand(connection, "PWR 0", regex)
	return err
}

// Weight retrieves the weight from the scale.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - measurement: the weight and unit of the measurement
//   - error: see DirectCommand()
func Weight(connection io.ReadWriteCloser) (Measurement, error) {
	regex := regexp.MustCompile(`S S +(-?[0-9]+\.[0-9]+) ([a-zA-Z]+)`)
	buf, err := DirectCommand(connection, "S", regex)
	if err != nil {
		return Measurement{}, err
	}

	result := regex.FindStringSubmatch(string(buf))
	weightValue, err := strconv.ParseFloat(result[1], 64)
	if err != nil {
		return Measurement{}, err
	}

	return Measurement{weightValue, result[2]}, nil
}

// WeightOnKey retrieves the weight from the scale when the transfer-key is pressed.
// The function will wait until the key has been pressed numMeasurements times, or until timeout is reached.
//
// Inputs:
//
//   - connection: the connection to the scale, created by Connect()
//
//   - numMeasurements: the number of measurements to take, or 0 for infinite
//
//   - timeout: max time.Duration for the function to be active, or 0 for infinite
//
//     Note: timeout and numMeasurements cannot both be infinite (0)
//
// Outputs:
//   - []Measurement: the weights and units of the measurements
//   - error: see DirectCommand()
func WeightOnKey(connection io.ReadWriteCloser, numMeasurements int, timeout time.Duration) ([]Measurement, error) {
	if timeout == 0 && numMeasurements == 0 {
		return []Measurement{}, fmt.Errorf("timeout and numMeasurements cannot both be infinite (0)")
	}
	if timeout == 0 {
		timeout = 1<<63 - 1 // MaxInt64 = 292 years
	}
	if numMeasurements == 0 {
		numMeasurements = int(math.Inf(1))
	}

	initRegex := regexp.MustCompile(`ST A`)
	_, err := DirectCommand(connection, "ST 1", initRegex)
	if err != nil {
		return []Measurement{}, err
	}

	weightRegex := regexp.MustCompile(`ST +(-?[0-9]+\.[0-9]+) ([a-zA-Z]+)`)
	// Read (until match or timeout)
	buf := make([]byte, 128)
	start := time.Now()
	i := 0
	var weightList []Measurement
	for i < numMeasurements && time.Since(start) < timeout {
		_, err = connection.Read(buf)
		if err != nil {
			return []Measurement{}, err
		}

		if weightRegex.Match(buf) {
			result := weightRegex.FindStringSubmatch(string(buf))
			weightValue, err := strconv.ParseFloat(result[1], 64)
			if err != nil {
				return []Measurement{}, err
			}

			weightList = append(weightList, Measurement{weightValue, result[2]})
			i++
		}
	}
	defer func() {
		stopRegex := regexp.MustCompile(`ST [AL]`) // ST L will be returned if the reading is already stopped by user interrupt
		_, _ = DirectCommand(connection, "ST 0", stopRegex)
	}()

	return weightList, nil
}

// Tare sets the current weight as the tare weight.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - []Measurement: the weight and unit of the measurement
//   - error: see DirectCommand()
func Tare(connection io.ReadWriteCloser) (Measurement, error) {
	regex := regexp.MustCompile(`T S +(-?[0-9]+\.[0-9]+) ([a-zA-Z]+)`)
	buf, err := DirectCommand(connection, "T", regex)
	if err != nil {
		return Measurement{}, err
	}
	result := regex.FindStringSubmatch(string(buf))
	weightValue, err := strconv.ParseFloat(result[1], 64)
	if err != nil {
		return Measurement{}, err
	}
	return Measurement{weightValue, result[2]}, nil
}

// GetDoorStatus retrieves the status of the Draft shield doors.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - string: the status of the doors, according to the status table
//   - error: see DirectCommand()
//
// Status table:
//
//	0: All draft shield doors closed
//	1: Right draft shield door(s) open
//	2: Left draft shield door(s) open
//	3: Top draft shield door open
//	4: Right and left draft shield doors open
//	5: All draft shield doors open
//	6: Right and top draft shield doors open
//	7: Left and top draft shield doors open
//	8: Error
//	9: Intermediate
func GetDoorStatus(connection io.ReadWriteCloser) (string, error) {
	regex := regexp.MustCompile(`WS`)
	buf, err := DirectCommand(connection, "WS ([0-9])", regex)
	if err != nil {
		return "", err
	}

	result := regex.FindStringSubmatch(string(buf))
	return result[1], nil
}

// CloseAllDoors closes all draft shield doors.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - error: see DirectCommand()
func CloseAllDoors(connection io.ReadWriteCloser) error {
	regex := regexp.MustCompile(`WS [AL]`) // WS L will be returned if the doors are already closed
	_, err := DirectCommand(connection, "WS 0", regex)
	return err
}

// OpenRightDoor opens the right draft shield door.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - error: see DirectCommand()
func OpenRightDoor(connection io.ReadWriteCloser) error {
	regex := regexp.MustCompile(`WS [AL]`) // WS L will be returned if the right door is already open
	_, err := DirectCommand(connection, "WS 1", regex)
	return err
}

// OpenLeftDoor opens the left draft shield door.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - error: see DirectCommand()
func OpenLeftDoor(connection io.ReadWriteCloser) error {
	regex := regexp.MustCompile(`WS [AL]`) // WS L will be returned if the left door is already open
	_, err := DirectCommand(connection, "WS 2", regex)
	return err
}

// Zero sets the current weight as the zero weight.
//
// Inputs:
//   - connection: the connection to the scale, created by Connect()
//
// Outputs:
//   - error: see DirectCommand()
func Zero(connection io.ReadWriteCloser) error {
	regex := regexp.MustCompile(`Z A`)
	_, err := DirectCommand(connection, "Z", regex)
	return err
}

// Measurement represents a measurement on the scale.
type Measurement struct {
	Weight float64
	Unit   string
}
