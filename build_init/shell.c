#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <pty.h>
#include <termios.h>
#include <fcntl.h>
#include <stdarg.h>
#include <sys/select.h>
#include <sys/wait.h>
#include <sys/io.h>
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


int main()
{
    iopl(3);
    int master;
    pid_t pid;
    struct termios terminal;
    terminal.c_lflag &= ~(ECHO | ECHONL);

    pid = forkpty(&master, NULL, &terminal, NULL);

    // Unable to fork
    if (pid < 0) {
        perror("at fork");
        return 1;
    }

    // child
    else if (pid == 0) {
        char *args[] = { NULL };

        // run the program
        execl("/term2048", "/term2048", (char *) NULL);
        perror("in child");
    }

    // parent
    else {
        while (1) {

            fd_set read_fd, write_fd, error_fd;

            // Clear sets
            FD_ZERO(&read_fd);
            FD_ZERO(&write_fd);
            FD_ZERO(&error_fd);

            // Add master file descriptor to fd_set
            FD_SET(master, &read_fd);
            // Add std in to read fd set
            FD_SET(STDIN_FILENO, &read_fd);

            // figure what is ready
            select(master+1, &read_fd, &write_fd, &error_fd, NULL);

            char input;
            char output;
            /*
            if (FD_ISSET(master, &read_fd))
            {
                if (read(master, &output, 1) != -1)
                    ttyputc(output);
                else
                    break;
            }

            if (FD_ISSET(STDIN_FILENO, &read_fd))
            {
                input = ttygetc();
                write(master, &input, 1);
            }
            */
            ttyputc('x');
            //printf("x");
        }
    }
    return 0;
}
