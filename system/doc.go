// Package system provides commands to run on a machine, independent of the mode
// of connection and the underlying OS.
//
// The supported operating systems are debian, ubuntu, centos, redhat. All the
// API is guranteed to work only on these systems.
//
// At the most basic level system provides functionality to run raw commands on
// the system. It mainly abstracts installing packages and running daemons.
//
// To run commands which are long running or produce a lot of output please use
// the RunBigCmd method.
package system
