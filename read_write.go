package main

import (
	"errors"
	"io"
)

func (f *FileSystem) Read(inode *Inode, offset int64, buffer []byte) (int64, error) {
	// Check if offset is within file size
	// if offset is after the end of file, return 0, nil
	if offset >= inode.Size {
		return 0, nil
	}

	// Find the index of block to read
	indexofBlock := offset / BLOCK_SIZE
	// calcualte offset in that block (using the offset argument)
	offsetInBlock := offset % BLOCK_SIZE

	// Calculate the number of bytes to read

	// This is minimum between the buffer length or file size (without "offset" bytes)
	to_read := min(int64(len(buffer)), inode.Size-offset)
	// loop: Read data from blocks
	for i := int64(0); i < to_read; {
		n, err := f.ReadFromBlock(Block(inode.Blocks[indexofBlock]), offsetInBlock, buffer)
		i += n
		if err != nil {
			return n, err
		}
		offsetInBlock = 0
		indexofBlock++
		buffer = buffer[n:]
	}
	return to_read, nil
}

func UpDivision(a, b int64) int64 {
	return (a + b - 1) / b
}

func (f *FileSystem) Write(inode *Inode, offset int64, buffer []byte) (int64, error) {
	// jump to the offset
	// write the number of bytes from the buffer
	// allocate blocks when needed

	// calculate the new size (offset + size)
	size := offset + int64(len(buffer))
	// check if it's not greater than maximum file
	if size > BLOCK_SIZE*DIRECT_LINKS {
		return -1, errors.New("size is greater that maximum files size")
	}

	// calcualte the number of blocks file occupies = old
	// calucalte the number of blocks new size occupies = new
	// while old <= new
	// 	    allocate block
	old := UpDivision(inode.Size, BLOCK_SIZE)
	new := UpDivision(size, BLOCK_SIZE)
	for old < new {
		block, err := f.AllocateBlock()
		if err != nil {
			return -1, err
		}
		inode.Blocks[old] = int64(block)
		old++
	}
	// find the index of block to write
	index := offset / BLOCK_SIZE
	// calcualte offset in that block (using the offset argument)
	offsetBlock := offset % BLOCK_SIZE

	// BLOCK_SIZE = 64
	// offset = 12     offsetBlock = 12
	// offset = 100    offsetBlock = 36
	// offset = 130    offsetBlock = 2
	n := int64(len(buffer))
	// write the data to block
	for len(buffer) != 0 {
		block := inode.Blocks[index]
		var err error
		buffer, err = f.WriteToBlock(Block(block), offsetBlock, buffer)
		if err != nil {
			return -1, err
		}
		index++
	}
	// go to the next block if needed, and repeat
	if size > inode.Size {
		inode.Size = size
	}
	err := f.WriteInode(inode)
	if err != nil {
		return -1, err
	}

	return n, nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (f *FileSystem) WriteToBlock(block Block, offset int64, buffer []byte) ([]byte, error) {
	// jump to the block + offset
	// write the buffer to block, but stop if block ends
	// return the bytes that were not written

	to_write := BLOCK_SIZE - offset
	location := f.Superblock.BlocksOffset + int64(block)*BLOCK_SIZE + offset
	_, err := f.File.Seek(location, io.SeekStart)
	if err != nil {
		return []byte{}, err
	}
	end := min(to_write, int64(len(buffer)))
	n, err := f.File.Write(buffer[:end])
	return buffer[n:], err

}

func (f *FileSystem) ReadFromBlock(block Block, offset int64, buffer []byte) (int64, error) {
	// jump to the block + offset
	// read the block to buffer, but stop if block ends
	// return the number of bytes that was written
	to_read := BLOCK_SIZE - offset
	location := f.Superblock.BlocksOffset + int64(block)*BLOCK_SIZE + offset
	_, err := f.File.Seek(location, io.SeekStart)
	if err != nil {
		return -1, err
	}
	end := min(to_read, int64(len(buffer)))
	n, err := f.File.Read(buffer[:end])
	if int64(n) != end {
		return int64(n), errors.New("ooooops")
	}
	return int64(n), err
}

func (f *FileSystem) Truncate(inode *Inode, size int64) error {
	// if newsize < inode.size
	//        reduce size (deallocate blocks)
	// if newsize > inode.size:
	//        allocate blocks (like in write)
	old := UpDivision(inode.Size, BLOCK_SIZE)
	new := UpDivision(size, BLOCK_SIZE)
	if size > inode.Size {
		for old < new {
			block, err := f.AllocateBlock()
			if err != nil {
				return err
			}
			inode.Blocks[old] = int64(block)
			old++
		}
	} else if size < inode.Size {
		for new < old {
			idx := old - 1
			err := f.SetBlockBitmapOffset(Block(inode.Blocks[idx]), FREE)
			if err != nil {
				return err
			}
			inode.Blocks[idx] = int64(0)
			old--
		}
	}
	inode.Size = size
	err := f.WriteInode(inode)
	return err
}
