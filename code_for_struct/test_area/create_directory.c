#include <stdio.h>
#include <sys/stat.h>
#include <errno.h>
#define DIR_EXISTS 1
#define DIR_CREATED 0

int create_directory(char *dir_path)
{
	if(!mkdir(dir_path,0755))
	{
		return DIR_CREATED;
	}
	else if(errno == EEXIST)
	{
		return DIR_EXISTS;
	}
	else
	{
		return -1;
	}
}

int main(int *argc, char *argv[])
{
	int ret_val = create_directory(argv[1]);
	if(ret_val == DIR_CREATED)
	{
		printf("directory has been created");
	}	
	else if (ret_val == DIR_EXISTS)
	{
		printf("directory already exists");
	}
	else 
	{
		printf("Unknown error occured");
	}
}
