shell:
	cc -o init shell.c -lutil
	find . -print | cpio -o -H newc > ../initramfs.linux_amd64.cpio

probe:
	cc -o init printforever.c
	find . -print | cpio -o -H newc > ../initramfs.linux_amd64.cpio
