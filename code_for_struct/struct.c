/*Headers section*/
#include <stdio.h>
#include <linux/limits.h>
#include <errno.h>
#include <sys/stat.h>
#include <unistd.h>

/*Macros section*/

//Test for existence.
#define F_OK 0
//Confirmation of existance of a file
#define FILE_EXISTS 1
//Confirmation of new creation of a file
#define FILE_CREATED 0
//error code for directory existance
#define EEXIST 17
//Confirmation of existance of a directory
#define DIR_EXISTS 1
//Confirmation of creation of a directory
#define DIR_CREATED 0


/*Global variable section*/

//RUN Variables

//variable used to tell the program whether to continue running or not 
int run = 1;
//variable used to tell the program whether to continue creating a structure of not once an operation is done
int run_create = 1;

/*Functions section*/

//TODO create the function which clears the terminal by priting a lot of new lines

int main()
{   
    while(run == 1)
    {
        //variable used to know what structure operation needs to be done
        int choice_of_struct = 0;

        //string used to store the path of presently operating directory [used when creating a structure]
        char path[PATH_MAX] = "\0";

        //TODO call the function which clears the terminal by priting a lot of new lines
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

                    //TODO call the function which clears the terminal by priting a lot of new lines
                    printf("1. Create a file");
                    printf("\n2. Create a directory");
                    printf("\n3. Create a script file");
                    printf("\n4. Move into a directory");
                    printf("\n5. Move to parent directory");
                    printf("\n6. Remove a file");
                    printf("\n7. Remove a directory");
                    printf("\n8. Rename a file");
                    printf("\n9. Renanme a directory");
                    printf("\n10. End making a structuren\n");
                    printf("\n\nEnter your choice : ");
                    scanf("%d",&choice_of_operation);

                    switch (choice_of_operation)
                    {
                        case 1:
                            //TODO Embed Create a file code
                            printf("\nfeature not available yet\n");                
                            break;
                        case 2:
                            //TODO Embed Create a directory code
                            printf("\nfeature not available yet\n");                
                            break;
                        case 3:
                            //TODO Embed Create file with script flag code here
                            printf("\nfeature not available yet\n");                
                            break;
                        case 4:
                            //TODO Update the path string
                            printf("\nfeature not available yet\n");                
                            break;
                        case 5:
                            //TODO Update the path string
                            printf("\nfeature not available yet\n");                
                            break;
                        case 6:
                            //TODO Create a function to Remove a file and embed it here
                            printf("\nfeature not available yet\n");                
                            break;
                        case 7:
                            //TODO Create a function to Remove a directory and embed it here
                            printf("\nfeature not available yet\n");                
                            break;
                        case 8:
                            //TODO Create a function to Rename a file and embed it here
                            printf("\nfeature not available yet\n");                
                            break;
                        case 9:
                            //TODO Create a function to Rename a directory and embed it here
                            printf("\nfeature not available yet\n");                
                            break;
                        case 10:
                            run_create = 0;
                            break;
                        default:
                            printf("\nfound invalid operation number, please enter a valid one\n");
                            break;
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