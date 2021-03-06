// display.go
package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/nsf/termbox-go"
)

func display() {
	termbox.Clear(termbox.ColorWhite, termbox.ColorBlack)
	headstr := fmt.Sprintf(" MariaDB Replication Monitor and Health Checker version %s ", repmgrVersion)
	if failover != "" {
		if interactive == false {
			headstr += " |  Mode: Auto Failover "
		} else {
			headstr += " |  Mode: Failover "
		}
	} else {
		headstr += " |  Mode: Switchover "
	}
	printfTb(0, 0, termbox.ColorWhite, termbox.ColorBlack|termbox.AttrReverse|termbox.AttrBold, headstr)
	printfTb(0, 5, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "%15s %6s %7s %12s %20s %20s %20s %6s %3s", "Slave Host", "Port", "Binlog", "Using GTID", "Current GTID", "Slave GTID", "Replication Health", "Delay", "RO")
	// Check Master Status and print it out to terminal. Increment failure counter if needed.
	err := master.refresh()
	if err != nil && err != sql.ErrNoRows && failCount < maxfail {
		failCount++
		tlog.Add(fmt.Sprintf("Master Failure detected! Retry %d/%d", failCount, maxfail))
		if failCount == maxfail {
			tlog.Add("Declaring master as failed")
			master.State = stateFailed
			master.CurrentGtid = "MASTER FAILED"
			master.BinlogPos = "MASTER FAILED"
		}
		termbox.Sync()
	}
	printfTb(0, 2, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "%15s %6s %41s %20s %12s", "Master Host", "Port", "Current GTID", "Binlog Position", "Strict Mode")
	printfTb(0, 3, termbox.ColorWhite, termbox.ColorBlack, "%15s %6s %41s %20s %12s", master.Host, master.Port, master.CurrentGtid, master.BinlogPos, master.Strict)
	vy = 6
	for _, slave := range slaves {
		slave.refresh()
		printfTb(0, vy, termbox.ColorWhite, termbox.ColorBlack, "%15s %6s %7s %12s %20s %20s %20s %6d %3s", slave.Host, slave.Port, slave.LogBin, slave.UsingGtid, slave.CurrentGtid, slave.SlaveGtid, slave.healthCheck(), slave.Delay.Int64, slave.ReadOnly)
		vy++
	}
	vy++
	for _, server := range servers {
		f := false
		if server.State == stateUnconn || server.State == stateFailed {
			if f == false {
				printfTb(0, vy, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "%15s %6s %41s %20s %12s", "Standalone Host", "Port", "Current GTID", "Binlog Position", "Strict Mode")
				f = true
				vy++
			}
			server.refresh()
			if server.State == stateFailed {
				server.CurrentGtid = "FAILED"
				server.BinlogPos = "FAILED"
			}
			printfTb(0, vy, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlack, "%15s %6s %41s %20s %12s", "Master Host", "Port", "Current GTID", "Binlog Position", "Strict Mode")
			printfTb(0, vy, termbox.ColorWhite, termbox.ColorBlack, "%15s %6s %41s %20s %12s", server.Host, server.Port, server.CurrentGtid, server.BinlogPos, server.Strict)
			vy++
		}

	}
	vy++
	if master.State != stateFailed {
		printTb(0, vy, termbox.ColorWhite, termbox.ColorBlack, " Ctrl-Q to quit, Ctrl-S to switchover")
	} else {
		printTb(0, vy, termbox.ColorWhite, termbox.ColorBlack, " Ctrl-Q to quit, Ctrl-F to failover")
	}
	vy = vy + 3
	tlog.Print()
	termbox.Flush()
	_, newlen := termbox.Size()
	if newlen > termlength {
		termlength = newlen
		tlog.len = termlength - 9 - (len(hostList) * 3)
		tlog.Extend()
	} else if newlen < termlength {
		termlength = newlen
		tlog.len = termlength - 9 - (len(hostList) * 3)
		tlog.Shrink()
	}
}

func printTb(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}

func printfTb(x, y int, fg, bg termbox.Attribute, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	printTb(x, y, fg, bg, s)
}

func logprint(msg ...interface{}) {
	stamp := fmt.Sprint(time.Now().Format("2006/01/02 15:04:05"))
	if logfile != "" {
		s := fmt.Sprint(stamp, " ", fmt.Sprintln(msg...))
		io.WriteString(logPtr, fmt.Sprint(s))
	}
	if tlog.len > 0 {
		tlog.Add(fmt.Sprintln(msg...))
		display()
	} else {
		log.Println(msg...)
	}
}

func logprintf(format string, args ...interface{}) {
	if logfile != "" {
		f := fmt.Sprintln(fmt.Sprint(time.Now().Format("2006/01/02 15:04:05")), format)
		io.WriteString(logPtr, fmt.Sprintf(f, args...))
	}
	if tlog.len > 0 {
		tlog.Add(fmt.Sprintf(format, args...))
		display()
	} else {
		log.Printf(format, args...)
	}
}
