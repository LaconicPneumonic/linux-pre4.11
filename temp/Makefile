shell:
	cc -static -o init shell.c -lutil
	find . -print | cpio -o -H newc > ../initramfs.linux_amd64.cpio

probe:
	cc -static -o init printforever.c
	find . -print | cpio -o -H newc > ../initramfs.linux_amd64.cpio
