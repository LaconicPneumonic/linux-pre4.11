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

int main(int argc, char *argv)
{
  iopl(3);

  while (1) {
    ttyputc(ttygetc());
  }

  exit(0);
}
