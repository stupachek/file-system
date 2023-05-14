package main

import "errors"

var ErrFileIsNotRegular error = errors.New("file is not regular")
var ErrDirIsNotEmpty error = errors.New("directory is not empty")
var ErrDelDot error = errors.New("can't delete \".\" or \"..\"")

type Stat struct {
	inode int64
	ftype FileType
	size  int64
	links int64
}

func (f *FileSystem) Create(dir int64, name string, ftype FileType) (int64, error) {
	directory, err := f.ReadInode(dir)
	if err != nil {
		return -1, err
	}
	if directory.fileType != DIRECTORY {
		return -1, ErrFileIsNotDir
	}
	file := Inode{}
	if ftype == REGULAR {
		file, err = f.AllocateInode()
		if err != nil {
			return -1, err
		}
		err = f.WriteInode(&file)
		if err != nil {
			return -1, err
		}

	} else {
		file, err = f.AllocateDirectory()
		if err != nil {
			return -1, err
		}
		err = f.AddFile(&file, ".", &file)
		if err != nil {
			return -1, err
		}
		err = f.AddFile(&file, "..", &directory)
		if err != nil {
			return -1, err
		}
	}

	err = f.AddFile(&directory, name, &file)
	if err != nil {
		return -1, err
	}
	return file.id, nil
}

func (f *FileSystem) List(dir int64) ([]Entry, error) {
	// read inode
	inode, err := f.ReadInode(dir)
	if err != nil {
		return nil, err
	}
	// read directory entries
	entry, err := f.ReadDirectory(&inode)
	return entry, err
}

func (f *FileSystem) Lookup(dir int64, name string) (int64, error) {
	// read inode
	// read directory entries
	entry, err := f.List(dir)
	if err != nil {
		return -1, err
	}
	// find the file with the given name and return the inode
	for _, file := range entry {
		if file.Name == name {
			return file.Inode, nil
		}
	}
	return -1, ErrFileNotFound
}

func (f *FileSystem) ReadFile(file int64, offset int64, buffer []byte) (int64, error) {
	// read the inode
	inode, err := f.ReadInode(file)
	if err != nil {
		return -1, err
	}
	// check that it's file
	if inode.fileType != REGULAR {
		return -1, ErrFileIsNotRegular
	}
	// read data
	return f.Read(&inode, offset, buffer)

}

func (f *FileSystem) WriteFile(file int64, offset int64, buffer []byte) (int64, error) {
	// read the inode
	inode, err := f.ReadInode(file)
	if err != nil {
		return -1, err
	}
	// check that it's file
	if inode.fileType != REGULAR {
		return -1, ErrFileIsNotRegular
	}
	// write data
	return f.Write(&inode, offset, buffer)
}

func (f *FileSystem) LinkFile(dir int64, name string, file int64) error {
	// read the dir
	// check that it's directory
	directory, err := f.ReadInode(dir)
	if err != nil {
		return err
	}
	if directory.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	// read the file, check that it's file
	fileForLink, err := f.ReadInode(file)
	if err != nil {
		return err
	}
	if fileForLink.fileType != REGULAR {
		return ErrFileIsNotRegular
	}
	// add the file to the directory
	err = f.AddFile(&directory, name, &fileForLink)
	return err
}

func (f *FileSystem) UnlinkFile(dir int64, name string) error {
	// forbid "." and ".."
	if name == "." || name == ".." {
		return ErrDelDot
	}
	// read the dir
	// check that it's directory
	directory, err := f.ReadInode(dir)
	if err != nil {
		return err
	}
	if directory.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	in, err := f.Lookup(dir, name)
	if err != nil {
		return err
	}
	file, err := f.ReadInode(in)
	if err != nil {
		return err
	}
	if file.fileType == DIRECTORY {
		entry, err := f.List(file.id)
		if err != nil {
			return err
		}
		if len(entry) != 2 {
			return ErrDirIsNotEmpty
		}
		_, err = f.RemoveFile(&file, ".")
		if err != nil {
			return err
		}
		_, err = f.RemoveFile(&file, "..")
		if err != nil {
			return err
		}

	}
	// remove the file
	file, err = f.RemoveFile(&directory, name)
	if err != nil {
		return err
	}
	// deallocate the inode, it the counter is 0
	if file.linkCount == 0 {
		err = f.DeallocateInode(&file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FileSystem) Stat(file int64) (Stat, error) {
	// read inode
	inode, err := f.ReadInode(file)
	if err != nil {
		return Stat{}, err
	}
	// fill struct
	return Stat{
		inode: inode.id,
		ftype: inode.fileType,
		size:  inode.Size,
		links: inode.linkCount,
	}, err
}
