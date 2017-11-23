// Package info provides the OSI presentation layer.
package info

import "errors"

// CommonAddr is a station address. Zero is not used.
// The width is controlled by Params.CommonAddrSize.
// See companion standard 101, subclause 7.2.4.
type CommonAddr uint16

var errAddrZero = errors.New("part5: common address 0 is not used")

// GlobalAddr is the broadcast address. Use is restricted
// to C_IC_NA_1, C_CI_NA_1, C_CS_NA_1 and C_RP_NA_1.
// When in 8-bit mode 255 is mapped to this value on the fly.
const GlobalAddr CommonAddr = 65535

// ObjAddr is the information object address.
// The width is controlled by Params.ObjAddrSize.
// See companion standard 101, subclause 7.2.5.
type ObjAddr uint

// Zero means that the address is irrelevant.
const IrrelevantAddr ObjAddr = 0

// SinglePoint is a measured value of a switch including quality
// descriptor flags Blocked, Substituted, NotTopical and Invalid.
// See companion standard 101, subclause 7.2.6.1.
type SinglePoint uint

// Two states with OK quality descriptor.
const (
	Off SinglePoint = iota
	On
)

// Split returns the state and the quality descriptor flags separated.
func (p SinglePoint) Split() (SinglePoint, QualDesc) {
	return p & 1, QualDesc(p & 0xf0)
}

// DoublePoint is a measured value of a determination aware switch including
// quality descriptor flags Blocked, Substituted, NotTopical and Invalid.
// See companion standard 101, subclause 7.2.6.2.
// See http://blog.iec61850.com/2009/04/why-do-we-need-single-point-and-double.html
type DoublePoint uint

// Four states with OK quality descriptor.
const (
	IndeterminateOrIntermediate DoublePoint = iota
	DeterminedOff
	DeterminedOn
	Indeterminate
)

// Split returns the state and the quality descriptor flags separated.
func (p DoublePoint) Split() (DoublePoint, QualDesc) {
	return p & 3, QualDesc(p & 0xf0)
}

// Quality descriptor flags attribute measured values.
// See companion standard 101, subclause 7.2.6.3.
type QualDesc uint

const (
	// Overflow marks whether the value is beyond a predefined range.
	Overflow = 1 << iota

	_ // reserve
	_ // reserve

	// TimeInvalid flags that the elapsed time was incorrectly acquired.
	// This attribute is only valid for events of protection equipment.
	// See companion standard 101, subclause 7.2.6.4.
	TimeInvalid

	// Blocked flags that the value is blocked for transmission; the
	// value remains in the state that was acquired before it was blocked.
	Blocked

	// Substituted flags that the value was provided by the input of
	// an operator (dispatcher) instead of an automatic source.
	Substituted

	// NotTopical flags that the most recent update was unsuccessful.
	NotTopical

	// Invalid flags that the value was incorrectly acquired.
	Invalid

	// OK means no flags, no problems.
	OK = 0
)

// StepPos is a value with transient state indication including quality descriptor.
// See companion standard 101, subclause 7.2.6.5.
type StepPos uint

// NewStepPos returns a new step position.
// Values out of range (-64, 63) oveflow silently.
func NewStepPos(value int, transient bool, q QualDesc) StepPos {
	p := StepPos(value & 0x7f)
	if transient {
		p |= 0x80
	}
	p |= StepPos(q << 8)
	return p
}

// Split returns the value in the range of (-64, 63) including wheather
// the equipment is transient state and the quality descriptor flags.
func (p StepPos) Split() (value int, transient bool, q QualDesc) {
	u := uint(p)
	if u&0x40 == 0 {
		// trim rest
		value = int(u & 0x3f)
	} else {
		// sign extend
		value = int(u) | (-1 &^ 0x3f)
	}
	transient = u&0x80 != 0
	q = QualDesc(u >> 8)
	return
}

// Normal is a 16-bit normalized value.
// See companion standard 101, subclause 7.2.6.6.
type Normal int16

// Float64 returns the actual value in the range of (-1, 1 − 2⁻¹⁵)
func (n Normal) Float64() float64 { return float64(n) / 32768 }

// SingleCmd is a singe command.
// See companion standard 101, subclause 7.2.6.15.
type SingleCmd struct{ Cmd }

// NewSingleCmd returns a new single command.
// The function panics when the qualifier exceeds range (0, 31).
func NewSingleCmd(state SinglePoint, qual uint, exec bool) SingleCmd {
	return SingleCmd{Cmd(state) | newCmd(qual, exec)}
}

// State returns the command's value.
func (c SingleCmd) State() SinglePoint { return SinglePoint(c.Cmd & 1) }

// DoubleCmd is a double command.
// See companion standard 101, subclause 7.2.6.16.
type DoubleCmd struct{ Cmd }

// NewDoubleCmd returns a new double command.
// The function panics when the qualifier exceeds range (0, 31).
func NewDoubleCmd(state DoublePoint, qual uint, exec bool) DoubleCmd {
	return DoubleCmd{Cmd(state) | newCmd(qual, exec)}
}

// State returns the command's value.
func (c DoubleCmd) State() DoublePoint { return DoublePoint(c.Cmd & 3) }

// StepCmd is a regulating step command.
// See companion standard 101, subclause 7.2.6.17.
type StepCmd struct{ Cmd }

// NewStepCmd returns a new regulating step command.
// The function panics when the qualifier exceeds range (0, 31).
func NewStepCmd(higher DoublePoint, qual uint, exec bool) StepCmd {
	return StepCmd{Cmd(higher) | newCmd(qual, exec)}
}

// Higher returns wheather the next step is higher (or lower).
func (c StepCmd) Higher() DoublePoint { return DoublePoint(c.Cmd & 3) }

// QualParam is the qualifier of parameter of measured values.
//
//	0: not used
//	1: threshold value
//	2: smoothing factor (filter time constant)
//	3: low limit for transmission of measured values
//	4: high limit for transmission of measured values
//	5‥31: reserved for standard definitions of this companion standard (compatible range)
//	32‥63: reserved for special use (private range)
//
// See companion standard 101, subclause 7.2.6.24.
type QualParam uint

const (
	// four standard kinds
	Threashold QualParam = iota + 1
	Smoothing
	LowLimit
	HighLimit

	// Change flags local parameter change.
	Change QualParam = 64

	// InOperation flags parameter operation.
	InOperation QualParam = 128
)

// Split returns the kind and the flags separated.
func (p QualParam) Split() (kind QualParam, change, inOperation bool) {
	return p & 63, p&Change != 0, p&InOperation != 0
}

// Cmd is a command including qualifier.
// See companion standard 101, subclause 7.2.6.26.
type Cmd uint

func newCmd(qual uint, exec bool) Cmd {
	if qual > 31 {
		panic("qualifier out of range")
	}
	if exec {
		return Cmd(qual << 2)
	}
	return Cmd((qual << 2) | 128)
}

// Qual returns the qualifier of command.
//
//	0: no additional definition
//	1: short pulse duration (circuit-breaker), duration determined by a system parameter in the outstation
//	2: long pulse duration, duration determined by a system parameter in the outstation
//	3: persistent output
//	4‥8: reserved for standard definitions of this companion standard
//	9‥15: reserved for the selection of other predefined functions
//	16‥31: reserved for special use (private range)
func (c Cmd) Qual() uint { return uint((c >> 2) & 31) }

// Exec returns whether the command executes (or selects).
// See section 5, subclause 6.8.
func (c Cmd) Exec() bool { return c&128 == 0 }

// SetpointCmd is the qualifier of a set-point command.
// See companion standard 101, subclause 7.2.6.39.
type SetpointCmd uint

// NewSetpointCmd returns a new set-point command.
// The function panics when the qualifier exceeds range (0, 127).
func NewSetpointCmd(qual uint, exec bool) SetpointCmd {
	if qual > 127 {
		panic("qualifier out of range")
	}
	if exec {
		return SetpointCmd(qual)
	}
	return SetpointCmd(qual | 128)
}

// Qual returns the qualifier of set-point command.
//
//	0: default
//	0‥63: reserved for standard definitions of this companion standard (compatible range)
//	64‥127: reserved for special use (private range)
func (c SetpointCmd) Qual() uint { return uint(c & 127) }

// Exec returns whether the command executes (or selects).
// See section 5, subclause 6.8.
func (c SetpointCmd) Exec() bool { return c&128 == 0 }