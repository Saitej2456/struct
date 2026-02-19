#include "backend.h"
#include <stdio.h>
#include <linux/limits.h>
#include <errno.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>
#include <string.h>
#include <stdlib.h>

#define PATH_MAX 4096

void generate_path(char *rpath, const char *cpath, const char *addon) {
    snprintf(rpath, PATH_MAX, "%s/%s", cpath, addon);
}

int existance_checker(const char *tpath) {
    if (access(tpath, F_OK) != -1) {
        struct stat statbuf;
        if (stat(tpath, &statbuf) != 0) return -2;
        return S_ISDIR(statbuf.st_mode) ? 0 : 1;
    }
    return -2;
}

int remove_dir_recursive(const char *dpath) {
    DIR *d = opendir(dpath);
    if (!d) return -1;
    struct dirent *dir;
    while ((dir = readdir(d)) != NULL) {
        if (strcmp(dir->d_name, ".") == 0 || strcmp(dir->d_name, "..") == 0) continue;
        char subpath[PATH_MAX];
        snprintf(subpath, PATH_MAX, "%s/%s", dpath, dir->d_name);
        if (existance_checker(subpath) == 0) {
            remove_dir_recursive(subpath);
        } else {
            remove(subpath);
        }
    }
    closedir(d);
    return rmdir(dpath);
}

int Bridge_CreateFile(char *path, char *name) {
    char fpath[PATH_MAX];
    generate_path(fpath, path, name);
    if (access(fpath, F_OK) == 0) return 0; 
    FILE *fptr = fopen(fpath, "w");
    if (fptr) { fclose(fptr); return 1; }
    return -1;
}

int Bridge_CreateScript(char *path, char *name) {
    char fpath[PATH_MAX];
    generate_path(fpath, path, name);
    if (access(fpath, F_OK) == 0) return 0;
    FILE *fptr = fopen(fpath, "w");
    if (fptr) {
        fprintf(fptr, "#!/bin/bash\n");
        fclose(fptr);
        chmod(fpath, 0755); 
        return 1;
    }
    return -1;
}

int Bridge_CreateDir(char *path, char *name) {
    char dpath[PATH_MAX];
    generate_path(dpath, path, name);
    if (mkdir(dpath, 0755) == 0) return 1;
    return (errno == EEXIST) ? 0 : -1;
}

int Bridge_Rename(char *full_path, char *new_name) {
    char parent[PATH_MAX];
    strcpy(parent, full_path);
    char *last_slash = strrchr(parent, '/');
    if (last_slash) *last_slash = '\0'; 
    
    char new_full_path[PATH_MAX];
    snprintf(new_full_path, PATH_MAX, "%s/%s", parent, new_name);
    return rename(full_path, new_full_path) == 0 ? 1 : -1;
}

int Bridge_Delete(char *full_path) {
    int type = existance_checker(full_path);
    if (type == 0) return remove_dir_recursive(full_path) == 0 ? 1 : -1;
    else if (type == 1) return remove(full_path) == 0 ? 1 : -1;
    return 0; 
}