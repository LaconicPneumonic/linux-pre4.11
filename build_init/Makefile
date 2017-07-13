all:
	cc -static -o init printforever.c
	find . -print | cpio -o -H newc > ../initramfs.linux_amd64.cpio 
