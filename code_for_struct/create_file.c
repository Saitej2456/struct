#include <stdio.h>
#include <unistd.h>
#include <string.h>
#define FILE_EXISTS 1
#define FILE_CREATED 0


int create_file(char *fpath)
{
	FILE *fptr;
	if(access(fpath,F_OK) == 0)
	{
		return FILE_EXISTS;
	}
	else
	{
		fopen(fpath,"w");
		return FILE_CREATED;
	}
}

int main(int argc, char *argv[])
{
	char hiphen = '-';
	char script_flag_sym = 's';
	if((argv[1])[0] == '-')
	{
		if((argv[1])[1] == 's')
		{
			if(create_file(argv[2]) == FILE_EXISTS)
			{
				return 1;
			}
			else
			{
				return 0;
			}
		}

	}
	if(create_file(argv[1]) == FILE_EXISTS)
	{
		return 1;
	}
	else
	{
		return 0;
	}
	
}
