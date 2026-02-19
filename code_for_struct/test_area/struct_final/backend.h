#ifndef BACKEND_H
#define BACKEND_H

int Bridge_CreateFile(char *path, char *name);
int Bridge_CreateDir(char *path, char *name);
int Bridge_CreateScript(char *path, char *name);
int Bridge_Rename(char *full_path, char *new_name);
int Bridge_Delete(char *full_path); 

#endif