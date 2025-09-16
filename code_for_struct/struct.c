/*Headers section*/
#include <stdio.h>
#include <linux/limits.h>
#include <errno.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>

/*Macros section*/

//Test for existence.
#define F_OK 0
//Confirmation of existance of a file
#define FILE_EXISTS 1
//Confimration of non existance of a file
#define FILE_NEXISTS -2
//Confirmation of new creation of a file
#define FILE_CREATED 0
//Info that the check is for file
#define FILE_C 1
//Info that the check is for directory
#define DIR_C 0
//Error code for directory existance
#define EEXIST 17
//Confirmation of existance of a directory
#define DIR_EXISTS 1
//Confirmation of creation of a directory
#define DIR_CREATED 0
//Movement representor [forward]
#define FORWARD 1 
//Movement representor [backward]
#define BACKWARD -1
//Confirmation of file
#define IS_FILE 1
//Confirmation of directory 
#define IS_DIR 0


/*Global variable section*/

//RUN Variables

//variable used to tell the program whether to continue running or not 
int run = 1;
//variable used to tell the program whether to continue creating a structure of not once an operation is done
int run_create = 1;

/*Functions section*/

//Utility functions

//function which will create the illusion of clearing the terminal by printing a lot of "\n" [new line charecters] 
void clear_terminal()
{
    printf("\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n");
}

void fancy_path(char * path)
{
    printf("\n");
    printf("<<<<< Curr path : ");
    printf("%s >>>>>\n",path);
}

//Functions core to the program

//function to check existance of a file or directory 
int existance_checker(char *tpath)
{
    DIR *dirptr;
    if (access (tpath, F_OK) != -1 ) 
    {
        if ((dirptr = opendir (tpath)) != NULL) 
        {
            closedir (dirptr); 
            return IS_DIR;
        } 
        else 
        {
            return IS_FILE; 
        }
    } 
    else 
    {
        return FILE_NEXISTS;  
    }

}

//function to update the path [move between directories]
void update_path(char *cpath, char *addon, int movement)
{
    if(movement == -1)
    {
        int lastbefore_backslash_index = 0;
        for(int i = 0 ; i < PATH_MAX; i++)
        {
            if(cpath[i] == '/')
            {
                if(cpath[i+1] != '\0')
                {
                    lastbefore_backslash_index = i;
                }
                else
                {
                    for(int j = lastbefore_backslash_index+1 ; j < NAME_MAX ; j++)
                    {
                        if(cpath[j] != '\0')
                        {
                            cpath[j] = '\0';
                        }
                        else
                        {
                            break;
                        }
                    }
    
                    break;
                }
            }
        }
    }

    else if (movement == 1)
    {
        for(int i = 0 ; i < PATH_MAX ; i++)
        {
            if(cpath[i] != '\0')
            {
                continue;
            }
            else
            {
                int k = i;
                for(int j = 0 ; j < PATH_MAX ; j++)
                {
                    if(addon[j] != '\0')
                    {
                        cpath[k] = addon[j];
                        k++;
                        continue;
                    }
                    else
                    {
                        cpath[k] = '/';
                        k++;
                        cpath[k] = addon[j];
                        break;
                    }
                }

                if(existance_checker(cpath) == -2)
                {
                    update_path(cpath,"\0",BACKWARD);
                    printf("\nNo such directory exists\n");
                }
                break;
            }
        }
    }
}

//function to generate a new path with current path and addon
int generate_path(char *rpath, char *cpath, char *addon)
{
    int index_of_null = 0;
    for(int i = 0 ; i <= PATH_MAX ; i++)
    {
        if(i == PATH_MAX)
        {
            index_of_null = i;
            rpath[i] = '\0';
            printf("\nPath size is overflowing...\ncancelling creation of file...\n");
            //TODO implement handling for file path exceding case 
            break;
        }
        if(cpath[i] == '\0')
        {
            index_of_null = i;
            rpath[i] = cpath[i];
            break;
        }
        else
        {
            rpath[i] = cpath[i];
            continue;
        }
    }

    int j = 0;
    for(int i = index_of_null; i <= PATH_MAX; i++)
    {
        if(i == PATH_MAX)
        {
            index_of_null = i;
            rpath[i] = '\0';
            printf("\npath size is overflowing...\ncancelling creation of file...\n");
            //TODO implement handling for file path exceding case 
            break;
        }
        if(addon[j] == '\0')
        {
            index_of_null = i;
            rpath[i] = addon[j];
            break;      
        }
        else
        {
            rpath[i] = addon[j];
            j++;
        }
    }

    return index_of_null;
}

//function to create a file
int create_file(char *cpath)
{
    //string which stores the name of the file to create the file
    char fname[NAME_MAX]="\0";

    //string which is used to store the path of the file to be created
    char fpath[PATH_MAX]="\0";
    
    clear_terminal();
    printf("Enter the name of the file : ");
    scanf(" %[^\n]%*c" ,fname);
    
    generate_path(fpath, cpath, fname);
    printf("\n%s",fpath);

	FILE *fptr;
	if(access(fpath,F_OK) == 0)
	{
        printf("\nfile already exists!\ncancelling file creation...\n");
		return FILE_EXISTS;
	}
	else
	{
		fopen(fpath,"w");
		return FILE_CREATED;
	}
}

//function to create a directory
int create_directory(char *cpath)
{
    //string which stores the name of the directory to create the directory
    char dname[NAME_MAX]="\0";

    //string which is used to store the path of the directory to be created
    char dpath[PATH_MAX]="\0";
    
    clear_terminal();
    printf("Enter the name of the directory : ");
    scanf(" %[^\n]%*c" ,dname);
    
    generate_path(dpath,cpath,dname);
    printf("\n%s",dpath);
    
	if(!mkdir(dpath,0755))
	{
		return DIR_CREATED;
	}
	else if(errno == EEXIST)
	{
        printf("\nDirectory already exists\n");
		return DIR_EXISTS;
	}
	else
	{
		return -1;
	}
}

//function to remove a file 
void remove_file(char *cpath)
{
    clear_terminal();
    char fpath[PATH_MAX]="\0";
    char fname[NAME_MAX]="\0";
    printf("Enter file name here : ");
    scanf(" %[^\n]%*c",fname);
    generate_path(fpath,cpath,fname);
    int ret_val = existance_checker(fpath);
    
    if( ret_val == IS_FILE)
    {
        remove(fpath);
    }
    else if(ret_val == FILE_NEXISTS)
    {
        printf("No such file exists");
    }
}

//function to remove a directory
void remove_dir(char *dpath)
{
    int ret_val = existance_checker(dpath);
    
    if(ret_val == FILE_NEXISTS)
    {
        printf("No such Directory exists\nCancelling request\n");
    }
    else if(ret_val == IS_DIR)
    {
        DIR *d;
        struct dirent *dir;
        d = opendir(dpath);
        if(d)
        {
            while((dir = readdir(d)) != NULL)
            {
                if(dir->d_name[0] == '.')
                {
                    if(dir->d_name[1] == '.')
                    {
                        if(dir->d_name[2] == '\0')
                        {
                            continue;
                        }    
                    }
                    else if(dir->d_name[1] == '\0')
                    {
                        continue;
                    }
                    else
                    {
                        //TODO hidden directory case
                    }
                }
                else
                {
                    //TODO normal case
                    char tpath_in[PATH_MAX]="\0";
                    char tpath_out[PATH_MAX]="\0";
                    generate_path(tpath_in,dpath,"\0");
                    update_path(tpath_in,dir->d_name,FORWARD);
                    generate_path(tpath_out,dpath,dir->d_name);
                    if (existance_checker(tpath_out) == IS_FILE)
                    {
                        remove(tpath_out);
                    }
                    else if(existance_checker(tpath_out) == IS_DIR)
                    {
                        remove_dir(tpath_in);
                    }
                }
            }            
            closedir(d);
            rmdir(dpath);
        }
    }
    if(ret_val == IS_FILE)
    {
        remove(dpath);
    }
}

//function to rename a file/directory
void rename_FD(char *cpath, int FoD)
{
    clear_terminal();
    char oldname[NAME_MAX]="\0";
    char oldpath[PATH_MAX]="\0";
    char newname[NAME_MAX]="\0";
    char newpath[PATH_MAX]="\0";
    if(FoD = IS_FILE)
    {
        printf("Enter old file name here : ");
        scanf(" %[^\n]%*c",oldname);
        printf("\n\nEnter new file name here : ");
        scanf(" %[^\n]%*c",newname);
    }
    if(FoD = IS_DIR)
    {
        printf("Enter old directory name here : ");
        scanf(" %[^\n]%*c",oldname);
        printf("\n\nEnter new directory name here : ");
        scanf(" %[^\n]%*c",newname);
    }
    generate_path(oldpath,cpath,oldname);
    generate_path(newpath,cpath,newname);
    rename(oldpath,newpath);
}


//TODO create a function which will take a message [string] along with the during it should be shown onto the terminal  
//TODO make a function which will go to the next instruction upon clicking enter

int main()
{   
    while(run == 1)
    {
        //variable used to know what structure operation needs to be done
        int choice_of_struct = 0;

        //string used to store the path of presently operating directory [used when creating a structure]
        char path[PATH_MAX] = "./structures/\0";

        clear_terminal();
        printf("1. Create a structure");
        printf("\n2. Use a structure");
        printf("\n3. Remove a structure");
        printf("\n4. Edit a structure");
        printf("\n5. Stop the program\n");
        printf("\n\nEnter your choice : ");  
        scanf("%d",&choice_of_struct);

        switch (choice_of_struct)
        {
            case 1:

                while(run_create == 1)
                {
                    //variable used to know which operation needs to be performed while creating a structure 
                    int choice_of_operation = 0;

                    // clear_terminal();
                    fancy_path(path);
                    printf("1. Create a file");
                    printf("\n2. Create a directory");
                    printf("\n3. Create a script file");
                    printf("\n4. Move into a directory");
                    printf("\n5. Move to parent directory");
                    printf("\n6. Remove a file");
                    printf("\n7. Remove a directory");
                    printf("\n8. Rename a file");
                    printf("\n9. Renanme a directory");
                    printf("\n10. List current path contents");
                    printf("\n11. End making a structure\n");
                    printf("\n\nEnter your choice : ");
                    scanf("%d",&choice_of_operation);

                    switch (choice_of_operation)
                    {
                        case 1:
                        {
                            create_file(path);
                            break;
                        }
                        case 2:
                        {
                            create_directory(path);
                            break;
                        }
                        case 3:
                        {
                            //TODO Embed Create file with script flag code here
                            printf("\nfeature not available yet\n");                
                            break;
                        }
                        case 4:
                        {
                            char tname[NAME_MAX]="\0";
                            scanf(" %[^\n]%*c",tname);
                            update_path(path,tname,FORWARD);
                            break;
                        }
                        case 5:
                        {
                            update_path(path,"\0",BACKWARD);
                            break;
                        }
                        case 6:
                        {
                            remove_file(path);
                            break;
                        }
                        case 7:
                        {
                            char dname[NAME_MAX]="\0";
                            clear_terminal();
                            printf("\nEnter the directory name :\n");
                            scanf(" %[^\n]%*c",dname);
                            char dpath[PATH_MAX]="\0";
                            generate_path(dpath,path,"\0");
                            update_path(dpath,dname,FORWARD);
                            remove_dir(dpath);
                            break;
                        }
                        case 8:
                        {
                            rename_FD(path,IS_FILE);          
                            break;
                        }
                        case 9:
                        {
                            rename_FD(path,IS_DIR);
                            break;
                        }
                        case 10:
                        {
                            DIR *d;
                            struct dirent *dir;
                            d = opendir(path);
                            if(d)
                            {
                                clear_terminal();
                                while((dir = readdir(d)) != NULL)
                                {
                                    printf("\n%s",dir->d_name);
                                }
                                char confirm;
                                scanf(" %c",&confirm);
                                closedir(d);
                                dir = NULL;
                            }
                            break;
                        }
                        case 11:
                        {
                            run_create = 0;
                            break;
                        }
                        default:
                        {
                            printf("\nfound invalid operation number, please enter a valid one\n");
                            break;
                        }
                    }
                }    
                break;
            case 2:
                //TODO write code for using the created structures
                printf("\nfeature not available yet\n");
                break;
            case 3:
                //TODO write code for removing created strctures 
                printf("\nfeature not available yet\n");                
                break;
            case 4:
                //TODO edit code for editing existing strctures
                printf("\nfeature not available yet\n");                
                break;
            case 5:
                printf("\n<<<< exiting the program >>>>\n");
                run = 0;                
                break;
            default:
                printf("\nfound invalid operation number, please enter a valid one\n");
                break;
        }
    }
    
    return 0;
}