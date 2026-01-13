/*Headers section*/
#include <stdio.h>
#include <linux/limits.h>
#include <errno.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>

//charecter string used for path genral use case [try to avoid highly inefficient everytime usage is done the whole string should be cleared]
char gen_pathstring[PATH_MAX] = "\0";

//function which will clear all the charecters that may currently exist in the "char gen_pathstring[PATH_MAX]"
void clear_pathstring()
{
    for(int i = 0 ; i < PATH_MAX; i++)
    {
        gen_pathstring[i] = '\0';
    }
}

int ansc_rel_path(char * cpath, int ancestor_number)
{
    //The array which stores the indexes of the backslashes it encountered. Expanded as [back_slash_index]
    int bsi[PATH_MAX];
    int bsi_pointer = 0;
    int required_backslash_index = 0;
    int lastbefore_backslash_index = 0;
    int fse = 0;
    for(int i = 0 ; i < PATH_MAX; i++)
    {
        if(cpath[i] == '/')
        {
            if(cpath[i+1] != '\0')
            {
                bsi[bsi_pointer] = i;
                bsi_pointer++;
                continue;
            }
            else
            {
                bsi[bsi_pointer] = i;
                lastbefore_backslash_index = i;
                fse=1;
                break;
            }
        }
        else if(cpath[i] == '\0')
        {
            bsi_pointer--;
            if(bsi_pointer == -1)
            {
                printf("no forward slashes found error");
                return -2;
            }
            lastbefore_backslash_index = bsi[bsi_pointer];
            fse=0;
            break;
        }
    }

    if(fse == 1)
    {
        //This is the index of bsi[PATH_MAX] which will be storing the actual index location of the required backslash in cpath string
        int req_sindex = bsi_pointer - ancestor_number;
        required_backslash_index = bsi[req_sindex];
        int k = 0;
        for(int i = required_backslash_index+1; i<lastbefore_backslash_index; i++)
        {
            gen_pathstring[k] = cpath[i];
            k++;
        }
    }
    else if (fse == 0)
    {
        //This is the index of bsi[PATH_MAX] which will be storing the actual index location of the required backslash in cpath string
        //
        //Note : ancestor-1 is taken because the bsi_pointer without forward slash at the end of cpath string, will be pointing to the forward slash which would be second last if a forward slash was there
        int req_sindex = bsi_pointer - (ancestor_number-1);
        required_backslash_index = bsi[req_sindex];
        int k = 0;
        for(int i = required_backslash_index+1; i<PATH_MAX; i++)
        {
            gen_pathstring[k] = cpath[i];
            k++;
        }
    }
} 

int main()
{
    ansc_rel_path("/hello/its/mer/",3);
    printf("%s",gen_pathstring);
    clear_pathstring();
    return 0;
}