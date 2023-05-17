package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/xandout/gorpl"
	"github.com/xandout/gorpl/action"
)

type Handler func(args ...interface{}) (interface{}, error)

func errorify(callback Handler) Handler {
	return func(args ...interface{}) (interface{}, error) {
		val, err := callback(args...)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		return val, err
	}
}

func NewRepl(fs *FileSystem) gorpl.Repl {
	exitAction := action.New("exit", errorify(func(args ...interface{}) (interface{}, error) {
		fmt.Println("Bye!")
		os.Exit(0)
		return nil, nil
	}))
	create := action.New("create", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need name")
		}
		name := args[0].(string)
		err := fs.CreateCmd(fs.Session.pwd, name)
		return nil, err
	}))
	ls := action.New("ls", errorify(func(args ...interface{}) (interface{}, error) {
		entry, err := fs.List(fs.Session.pwd)
		if err != nil {
			return nil, err
		}
		fmt.Println("inode\tname")
		for _, en := range entry {
			fmt.Printf("%v\t%s\n", en.Inode, en.Name)
		}
		return nil, nil
	}))
	link := action.New("link", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("need from and to")
		}
		from := args[0].(string)
		to := args[1].(string)
		err := fs.LinkCmd(fs.Session.pwd, from, to)
		return nil, err
	}))
	unlink := action.New("unlink", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need name")
		}
		name := args[0].(string)
		err := fs.UnlinkCmd(fs.Session.pwd, name)
		return nil, err
	}))
	truncate := action.New("truncate", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("need name and size ")
		}
		name := args[0].(string)
		sizeSrt := args[1].(string)
		size, err := strconv.Atoi(sizeSrt)
		if err != nil {
			return nil, errors.New("size should be int")
		}
		err = fs.TruncateCmd(fs.Session.pwd, name, int64(size))
		return nil, err
	}))
	stat := action.New("stat", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need name")
		}
		name := args[0].(string)
		stat, err := fs.StatCmd(fs.Session.pwd, name)
		if err != nil {
			return nil, err
		}

		ftype := "-"

		switch stat.ftype {
		case DIRECTORY:
			ftype = "d"
		case REGULAR:
			ftype = "r"
		case SYMLINK:
			ftype = fmt.Sprintf("s (%s)", stat.symlink)
		default:
			ftype = "-"
		}

		fmt.Printf("inode:\t%v\n", stat.inode)
		fmt.Printf("ftype:\t%s\n", ftype)
		fmt.Printf("size:\t%v\n", stat.size)
		fmt.Printf("links:\t%v\n", stat.links)
		return nil, nil
	}))
	open := action.New("open", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need name")
		}
		name := args[0].(string)
		fkay, err := fs.OpenCmd(fs.Session.pwd, name)
		if err != nil {
			return nil, err
		}
		fmt.Printf("fd = %v\n", fkay)
		return nil, err
	}))
	write := action.New("write", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, errors.New("need fd and data")
		}
		fd := args[0].(string)
		data := ""
		for _, str := range args[1:] {
			subStr := str.(string)
			data += subStr
			data += " "
		}
		err := fs.WriteCmd(Fkey(fd), data)
		return nil, err
	}))
	read := action.New("read", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("need fd and length")
		}
		fd := args[0].(string)
		lengthStr := args[1].(string)
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, errors.New("length should be int")
		}
		string, err := fs.ReadCmd(Fkey(fd), int64(length))
		if err != nil {
			return nil, err
		}
		fmt.Println(string)
		return nil, nil
	}))
	seek := action.New("seek", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("need fd and offset")
		}
		fd := args[0].(string)
		offsetStr := args[1].(string)
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, errors.New("length should be int")
		}
		err = fs.SeekCmd(Fkey(fd), int64(offset))
		return nil, err
	}))
	close := action.New("close", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need fd ")
		}
		fd := args[0].(string)
		err := fs.CloseCmd(Fkey(fd))
		return nil, err
	}))
	mkfs := action.New("mkfs", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need n")
		}
		nStr := args[0].(string)
		n, err := strconv.Atoi(nStr)
		if err != nil {
			return nil, errors.New("n should be int")
		}
		f, err := NewFileSystem(int64(n), "fs")
		if err != nil {
			return nil, err
		}
		fs = &f
		return nil, err
	}))
	mkdir := action.New("mkdir", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need name")
		}
		name := args[0].(string)
		err := fs.MkdirCmd(fs.Session.pwd, name)
		return nil, err
	}))
	cd := action.New("cd", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need path")
		}
		path := args[0].(string)
		err := fs.CdCmd(fs.Session.pwd, path)
		return nil, err
	}))
	rmdir := action.New("rmdir", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("need name")
		}
		name := args[0].(string)
		err := fs.RmdirCmd(fs.Session.pwd, name)
		return nil, err
	}))
	symlink := action.New("symlink", errorify(func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("need from and to")
		}
		from := args[0].(string)
		to := args[1].(string)
		err := fs.SymlinkCmd(fs.Session.pwd, from, to)
		return nil, err
	}))
	repl := gorpl.New(";")
	repl.AddAction(*exitAction)
	repl.AddAction(*create)
	repl.AddAction(*ls)
	repl.AddAction(*link)
	repl.AddAction(*unlink)
	repl.AddAction(*truncate)
	repl.AddAction(*stat)
	repl.AddAction(*open)
	repl.AddAction(*write)
	repl.AddAction(*read)
	repl.AddAction(*seek)
	repl.AddAction(*close)
	repl.AddAction(*mkfs)
	repl.AddAction(*mkdir)
	repl.AddAction(*cd)
	repl.AddAction(*rmdir)
	repl.AddAction(*symlink)
	return repl
}
