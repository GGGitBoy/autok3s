package hosts

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

type Tunnel struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Modes  ssh.TerminalModes
	Term   string
	Height int
	Weight int

	err  error
	conn *ssh.Client
	cmd  *bytes.Buffer
}

func (t *Tunnel) Close() error {
	return t.conn.Close()
}

func (t *Tunnel) Cmd(cmd string) *Tunnel {
	if t.cmd == nil {
		t.cmd = bytes.NewBufferString(cmd + "\n")
	}

	_, err := t.cmd.WriteString(cmd + "\n")
	if err != nil {
		t.err = err
	}

	return t
}

func (t *Tunnel) Terminal() error {
	session, err := t.conn.NewSession()
	if err != nil {
		return err
	}

	defer func() {
		_ = session.Close()
	}()

	if t.Stdin == nil {
		session.Stdin = os.Stdin
	} else {
		session.Stdin = t.Stdin
	}
	if t.Stdout == nil {
		session.Stdout = os.Stdout
	} else {
		session.Stdout = t.Stdout
	}
	if t.Stderr == nil {
		session.Stderr = os.Stderr
	} else {
		session.Stderr = t.Stderr
	}

	term := os.Getenv("TERM")
	if term == "" {
		t.Term = "xterm-256color"
	}
	t.Height = 40
	t.Weight = 80
	t.Modes = ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty(t.Term, t.Height, t.Weight, t.Modes); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return err
	}

	if err := session.Wait(); err != nil {
		return err
	}

	return nil
}

func (t *Tunnel) Run() error {
	if t.err != nil {
		return t.err
	}

	return t.executeCommands()
}

func (t *Tunnel) SetStdio(stdout, stderr io.Writer) *Tunnel {
	t.Stdout = stdout
	t.Stderr = stderr
	return t
}

func (t *Tunnel) executeCommands() error {
	for {
		cmd, err := t.cmd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := t.executeCommand(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (t *Tunnel) executeCommand(cmd string) error {
	session, err := t.conn.NewSession()
	if err != nil {
		return err
	}

	defer func() {
		_ = session.Close()
	}()

	session.Stdout = t.Stdout
	session.Stderr = t.Stderr

	if err := session.Run(cmd); err != nil {
		return err
	}

	return nil
}
