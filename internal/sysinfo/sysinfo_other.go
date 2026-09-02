//go:build !linux && !darwin

package sysinfo

func load1() float64      { return 0 }
func totalMemGB() float64 { return 0 }
func freeMemGB() float64  { return 0 }
func probeAccel() accel   { return accel{accel: "none"} }
func linkOf(string) Link  { return Link{} }
