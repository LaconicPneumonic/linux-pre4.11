#include <errno.h>
#include <string.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/io.h>
#include <sys/stat.h>
#include <sys/types.h>

void ttyputc(unsigned char c)
{
  while((inb(0x3fd) & 0x40) == 0);
  outb(c, 0x3f8);
}

unsigned char ttygetc(void)
{
  while((inb(0x3fd) & 0x1) == 0);
  return inb(0x3f8);
}

void ttyputs(char *c)
{
  while(*c) {
    if (*c == '\n') ttyputc('\r');
    ttyputc(*c);
    c++;
  }
}

void tty_printf(const char *format, ...)
{

  char out_buf[128];

  va_list arg_list;
  va_start(arg_list, format);

  vsprintf(out_buf, format, arg_list);
  ttyputs(out_buf);
}

void probe(int fd)
{

  struct stat fileStat;
  int success = fstat(fd, &fileStat);

  if (success != -1) {

    tty_printf("---------------------------\n");
    tty_printf("File Size: \t\t%d bytes\n",fileStat.st_size);
    tty_printf("Number of Links: \t%d\n",fileStat.st_nlink);
    tty_printf("File inode: \t\t%d\n",fileStat.st_ino);

    tty_printf("File Permissions: \t");
    tty_printf( (S_ISDIR(fileStat.st_mode)) ? "d" : "-");
    tty_printf( (fileStat.st_mode & S_IRUSR) ? "r" : "-");
    tty_printf( (fileStat.st_mode & S_IWUSR) ? "w" : "-");
    tty_printf( (fileStat.st_mode & S_IXUSR) ? "x" : "-");
    tty_printf( (fileStat.st_mode & S_IRGRP) ? "r" : "-");
    tty_printf( (fileStat.st_mode & S_IWGRP) ? "w" : "-");
    tty_printf( (fileStat.st_mode & S_IXGRP) ? "x" : "-");
    tty_printf( (fileStat.st_mode & S_IROTH) ? "r" : "-");
    tty_printf( (fileStat.st_mode & S_IWOTH) ? "w" : "-");
    tty_printf( (fileStat.st_mode & S_IXOTH) ? "x" : "-");
    tty_printf("\n\n");

    tty_printf("The file %s a symbolic link\n\n", (S_ISLNK(fileStat.st_mode)) ? "is" : "is not");
    tty_printf("The mode is %x\n", fileStat.st_mode);
  } else {
    int error_number = errno;
    char * errsv = strerror(error_number);
    tty_printf("strerror: %s\n", errsv);
    tty_printf("errno: %x\n", error_number);

  }
}

void probe_filename(char *path, char *message, int length)
{
  int fd = open(path, O_RDWR, 0);

  if (fd < 0 ) {
    tty_printf("%s ERROR: %s\n", path,  strerror(errno));
  } else {

    tty_printf("%s FD: %d\n", path, fd);

    int write_ret = write(fd, message, length);

    if (write_ret < 10) {
      tty_printf("%s WRITE ERROR: %s\n", path, strerror(errno));
    }
  }
}

int main(int argc, char *argv)
{
  static char buf[128];
  iopl(3);
  outb('a', 0x3f8);
  outb('a', 0x3f8);
  outb('a', 0x3f8);
  outb('a', 0x3f8);
  ttyputs("\n");
  int i, j;
  for(i = 0; i < 8; i++) {
    for(j = 0; j < 8; j++) {
      sprintf(buf, "%1x", j);
      ttyputs(buf);
    }
    ttyputs("\n");
  }
  sprintf(buf, "std input\n");
  ttyputs(buf);
  probe(0);

  sprintf(buf, "std output\n");
  ttyputs(buf);
  probe(1);

  sprintf(buf, "std error\n");
  ttyputs(buf);
  probe(2); 

  int write_ret = write(1, "HI\n", 3);

  if (write_ret < 3) {
    tty_printf("STDOUT WRITE ERROR: %s\n", strerror(errno));
  }
  write_ret = write(2, "XX\n", 3);

  if (write_ret < 3) {
    tty_printf("STDERROR WRITE ERROR: %s\n", strerror(errno));
  }

  probe_filename("/dev/ttyS0", "HI\n", 3);
  probe_filename("/dev/ttyUSB0", "YO\n", 3);

  while (1) {
    ttyputc(ttygetc());
  }

  exit(0);
}
