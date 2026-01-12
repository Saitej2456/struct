#include <stdio.h>

// function to copy a file from one place to another place 
void copy_file(char *destpath, char *srcpath)
{
    char buffer[512] = "\0";
    size_t buffer_size = sizeof(buffer)/sizeof(buffer[0]);
    size_t bytes_read = 0;

    FILE *src_file = fopen(srcpath,"r");
    if(src_file == NULL)
    {
        perror("Source file opening failed");
    }
    FILE *dest_file = fopen(destpath,"w+");
    if(dest_file == NULL)
    {
        perror("Destination file opening failed");
    }

    while((bytes_read=fread(buffer,sizeof(char),buffer_size,src_file))>0)
    {
        fwrite(buffer, sizeof(char), bytes_read, dest_file);
    }

    fclose(src_file);
    fclose(dest_file);

    src_file = dest_file = NULL;
}

int main ()
{
    copy_file("./testtt","./test.c");
    return 0;
}