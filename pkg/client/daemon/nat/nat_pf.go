// +build darwin

package nat

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	ppf "github.com/datawire/pf"
	"github.com/datawire/telepresence2/pkg/client"
)

type Translator struct {
	commonTranslator
	dev   *ppf.Handle
	token string
}

func pf(c context.Context, args []string, stdin string) error {
	dlog.Debugf(c, "running %s", client.ShellString("pfctl", args))
	cmd := exec.Command("pfctl", args...)
	cmd.Stdin = strings.NewReader(stdin)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}

func splitPorts(portspec string) (result []string) {
	for _, part := range strings.Split(portspec, ",") {
		result = append(result, strings.TrimSpace(part))
	}
	return
}

func fmtDest(a Address) (result []string) {
	ports := splitPorts(a.Port)

	if len(ports) == 0 {
		result = append(result, fmt.Sprintf("proto %s to %s", a.Proto, a.IP))
	} else {
		for _, port := range ports {
			addr := fmt.Sprintf("proto %s to %s", a.Proto, a.IP)
			if port != "" {
				addr += fmt.Sprintf(" port %s", port)
			}

			result = append(result, addr)
		}
	}

	return
}

func (t *Translator) rules() string {
	if t.dev == nil {
		return ""
	}

	entries := t.sorted()

	result := ""
	for _, entry := range entries {
		dst := entry.Destination
		for _, addr := range fmtDest(dst) {
			result += "rdr pass on lo0 inet " + addr + " -> 127.0.0.1 port " + entry.Port + "\n"
		}
	}

	result += "pass out quick inet proto tcp to 127.0.0.1/32\n"

	for _, entry := range entries {
		dst := entry.Destination
		for _, addr := range fmtDest(dst) {
			result += "pass out route-to lo0 inet " + addr + " keep state\n"
		}
	}

	return result
}

var actions = []ppf.Action{ppf.ActionPass, ppf.ActionRDR}

func (t *Translator) Enable(c context.Context) error {
	var err error
	t.dev, err = ppf.Open()
	if err != nil {
		return err
	}

	for _, action := range actions {
		var rule ppf.Rule
		err = rule.SetAnchorCall(t.Name)
		if err != nil {
			return err
		}
		rule.SetAction(action)
		rule.SetQuick(true)
		err = t.dev.PrependRule(rule)
		if err != nil {
			return err
		}
	}

	_ = pf(c, []string{"-a", t.Name, "-F", "all"}, "")

	// XXX: blah, this generates a syntax error, but also appears
	// necessary to make anything work. I'm guessing there is some
	// sort of side effect, like it is clearing rules or
	// something, although notably loading an empty ruleset
	// doesn't seem to work, it has to be a syntax error of some
	// kind.
	_ = pf(c, []string{"-f", "/dev/stdin"}, "pass on lo0")
	_ = pf(c, []string{"-a", t.Name, "-f", "/dev/stdin"}, t.rules())

	output, err := dexec.CommandContext(c, "pfctl", "-E").CombinedOutput()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "Token" {
			t.token = strings.TrimSpace(parts[1])
			break
		}
	}

	if t.token == "" {
		return errors.New("unable to parse token")
	}
	return nil
}

func (t *Translator) Disable(c context.Context) error {
	_ = _pf("-X", t.token)

	if t.dev != nil {
		for _, action := range actions {
		OUTER:
			for {
				rules, err := t.dev.Rules(action)
				if err != nil {
					return err
				}

				for _, rule := range rules {
					if rule.AnchorCall() == t.Name {
						dlog.Debugf(c, "removing rule: %v", rule)
						err = t.dev.RemoveRule(rule)
						if err != nil {
							return err
						}
						continue OUTER
					}
				}
				break
			}
		}
	}

	_ = pf(c, []string{"-a", t.Name, "-F", "all"}, "")
	return nil
}

func (t *Translator) ForwardTCP(c context.Context, ip, port, toPort string) error {
	return t.forward(c, "tcp", ip, port, toPort)
}

func (t *Translator) ForwardUDP(c context.Context, ip, port, toPort string) error {
	return t.forward(c, "udp", ip, port, toPort)
}

func (t *Translator) forward(c context.Context, protocol, ip, port, toPort string) error {
	t.clear(protocol, ip, port)
	t.Mappings[Address{protocol, ip, port}] = toPort
	return pf(c, []string{"-a", t.Name, "-f", "/dev/stdin"}, t.rules())
}

func (t *Translator) ClearTCP(c context.Context, ip, port string) error {
	t.clear("tcp", ip, port)
	return pf(c, []string{"-a", t.Name, "-f", "/dev/stdin"}, t.rules())
}

func (t *Translator) ClearUDP(c context.Context, ip, port string) error {
	t.clear("udp", ip, port)
	return pf(c, []string{"-a", t.Name, "-f", "/dev/stdin"}, t.rules())
}

func (t *Translator) clear(protocol, ip, port string) {
	delete(t.Mappings, Address{protocol, ip, port})
}

func (t *Translator) GetOriginalDst(conn *net.TCPConn) (rawaddr []byte, host string, err error) {
	remote := conn.RemoteAddr().(*net.TCPAddr)
	local := conn.LocalAddr().(*net.TCPAddr)
	addr, port, err := t.dev.NatLook(remote.IP.String(), remote.Port, local.IP.String(), local.Port)
	if err != nil {
		return
	}

	return nil, fmt.Sprintf("%s:%d", addr, port), nil
}
