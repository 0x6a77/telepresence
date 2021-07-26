package dpipe

import (
	"context"
	"io"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"golang.org/x/sys/windows"
)

func waitCloseAndKill(ctx context.Context, cmd *dexec.Cmd, peer io.Closer, closing *int32, killTimer **time.Timer) {
	<-ctx.Done()

	*killTimer = &time.Timer{} // Dummy timer since there's no correspondence to a hard kill
	atomic.StoreInt32(closing, 1)

	_ = peer.Close()

	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		dlog.Errorf(ctx, "Received %v trying to enumerate processes...", err)
	}
	dlog.Infof(ctx, "Process counting expedition")
	pids := []int{}
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(windows.ProcessEntry32{}))
	err = windows.Process32First(h, &pe)
	for err == nil {
		if pe.ParentProcessID == uint32(cmd.Process.Pid) {
			dlog.Debugf(ctx, "Found process %+v", pe)
			pids = append(pids, int(pe.ProcessID))
		}
		err = windows.Process32Next(h, &pe)
	}
	err = windows.CloseHandle(h)
	pids = append(pids, cmd.Process.Pid)
	if err != nil {
		dlog.Errorf(ctx, "Error %v while closing handle", err)
	}

	// This kills the process and any child processes that it has started. Very important when
	// killing sshfs-win since it starts a cygwin sshfs process that must be killed along with it
	for _, pid := range pids {
		_ = dexec.CommandContext(ctx, "taskkill", "/F", "/PID", strconv.Itoa(pid)).Run()
	}
	dlog.Infof(ctx, "Process %d and its children have been killed", cmd.Process.Pid)
}
