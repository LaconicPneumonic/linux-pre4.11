#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <pty.h>
#include <termios.h>
#include <fcntl.h>
#include <stdarg.h>
#include <sys/mount.h>
#include <sys/select.h>
#include <sys/wait.h>
#include <sys/io.h>
#include <sys/types.h>
#include <sys/stat.h>

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

void probe(char *name)
{

  struct stat fileStat;
  int success = stat(name, &fileStat);

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

int main()
{
    iopl(3);
    int master;
    pid_t pid;
    struct termios terminal;
    terminal.c_lflag &= ~(ECHO | ECHONL);
    char buf[128];
    // mount neccesary file systems
    int mount_error = mount("devpts", "/dev/pts/", "devpts", MS_MGC_VAL, NULL);
    mount("sysfs", "/sys", "sysfs", MS_MGC_VAL, NULL);
    mount("proc", "/proc", "proc", MS_MGC_VAL, NULL);
    
    if (mount_error < 0) {
        tty_printf("PID: %d, MASTER: %d, PTY NAME: %x", pid, master, buf);
        perror("at fork");
        return 1;
    }
    pid = forkpty(&master, buf, &terminal, NULL);
    // Unable to fork
    if (pid < 0) {
        tty_printf("PID: %d, MASTER: %d, PTY NAME: %x", pid, master, buf);
        perror("at fork");
        return 1;
    }

    // child
    else if (pid == 0) {
        char *args[] = { NULL };

        // run the program
        execl("bash-static", "bash-static", (char *) NULL);
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
            char output[128];
            
            if (FD_ISSET(master, &read_fd))
            {
                if (read(master, output, 128) != -1)
                    ttyputs(output);
                else
                    break;
            }

            if (FD_ISSET(STDIN_FILENO, &read_fd))
            {
                input = ttygetc();
                ttyputc(input);
                write(master, &input, 1);
            }
        }
    }
    return 0;
}
