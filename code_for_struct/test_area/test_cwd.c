#include <stdio.h>
#include <unistd.h>
#include <linux/limits.h>

int main()
{
    char CWD[PATH_MAX] = "\0";
    if((getcwd(CWD,sizeof(CWD)) != NULL))
    {
        printf("%s",CWD);
    }
    else
    {
        perror("error in getcwd");
    }
}