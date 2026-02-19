#ifndef BACKEND_H
#define BACKEND_H

// Core Bridge Functions for Go
int Bridge_CreateFile(char *path, char *name);
int Bridge_CreateDir(char *path, char *name);
int Bridge_CreateScript(char *path, char *name);
int Bridge_Rename(char *full_path, char *new_name);
int Bridge_Delete(char *full_path); // Handles both files and directories
int Bridge_CopyStruct(char *src_struct_path, char *dest_path);

#endif