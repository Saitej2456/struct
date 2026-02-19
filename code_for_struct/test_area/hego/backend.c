#include "backend.h"
#include <stdio.h>
#include <linux/limits.h>
#include <errno.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>
#include <string.h>
#include <stdlib.h>

// --- HELPER MACROS ---
#define PATH_MAX 4096
#define NAME_MAX 255

// --- HELPER FUNCTIONS ---

// Generates full path: rpath = cpath + "/" + addon
void generate_path(char *rpath, const char *cpath, const char *addon) {
    snprintf(rpath, PATH_MAX, "%s/%s", cpath, addon);
}

// Check if path is File (1), Dir (0), or Doesn't Exist (-2)
int existance_checker(const char *tpath) {
    if (access(tpath, F_OK) != -1) {
        struct stat statbuf;
        if (stat(tpath, &statbuf) != 0) return -2;
        return S_ISDIR(statbuf.st_mode) ? 0 : 1;
    }
    return -2;
}

// Recursive Delete
int remove_dir_recursive(const char *dpath) {
    DIR *d = opendir(dpath);
    if (!d) return -1;

    struct dirent *dir;
    while ((dir = readdir(d)) != NULL) {
        if (strcmp(dir->d_name, ".") == 0 || strcmp(dir->d_name, "..") == 0) continue;

        char subpath[PATH_MAX];
        snprintf(subpath, PATH_MAX, "%s/%s", dpath, dir->d_name);

        if (existance_checker(subpath) == 0) { // Is Directory
            remove_dir_recursive(subpath);
        } else { // Is File
            remove(subpath);
        }
    }
    closedir(d);
    return rmdir(dpath);
}

// Recursive Copy
void copy_dir_recursive(const char *dest_path, const char *src_path) {
    mkdir(dest_path, 0755);
    DIR *d = opendir(src_path);
    if (!d) return;

    struct dirent *dir;
    while ((dir = readdir(d)) != NULL) {
        if (strcmp(dir->d_name, ".") == 0 || strcmp(dir->d_name, "..") == 0) continue;

        char src_sub[PATH_MAX], dest_sub[PATH_MAX];
        snprintf(src_sub, PATH_MAX, "%s/%s", src_path, dir->d_name);
        snprintf(dest_sub, PATH_MAX, "%s/%s", dest_path, dir->d_name);

        if (existance_checker(src_sub) == 0) { // Dir
            copy_dir_recursive(dest_sub, src_sub);
        } else { // File
            FILE *src_f = fopen(src_sub, "rb");
            FILE *dest_f = fopen(dest_sub, "wb");
            if (src_f && dest_f) {
                char buf[1024];
                size_t n;
                while ((n = fread(buf, 1, sizeof(buf), src_f)) > 0) {
                    fwrite(buf, 1, n, dest_f);
                }
            }
            if (src_f) fclose(src_f);
            if (dest_f) fclose(dest_f);
        }
    }
    closedir(d);
}

// --- BRIDGE FUNCTIONS (CALLED BY GO) ---

int Bridge_CreateFile(char *path, char *name) {
    char fpath[PATH_MAX];
    generate_path(fpath, path, name);
    if (access(fpath, F_OK) == 0) return 0; // Exists
    FILE *fptr = fopen(fpath, "w");
    if (fptr) {
        fclose(fptr);
        return 1; // Success
    }
    return -1; // Fail
}

int Bridge_CreateScript(char *path, char *name) {
    char fpath[PATH_MAX];
    generate_path(fpath, path, name);
    // Ensure .sh extension if missing (handled in Go mostly, but safety here)
    if (access(fpath, F_OK) == 0) return 0;
    FILE *fptr = fopen(fpath, "w");
    if (fptr) {
        fprintf(fptr, "#!/bin/bash\n"); // Add hashbang
        fclose(fptr);
        chmod(fpath, 0755); // Make executable
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
    // We need to isolate the parent directory from full_path
    char parent[PATH_MAX];
    strcpy(parent, full_path);
    char *last_slash = strrchr(parent, '/');
    if (last_slash) *last_slash = '\0'; // Cut off the old name
    
    char new_full_path[PATH_MAX];
    snprintf(new_full_path, PATH_MAX, "%s/%s", parent, new_name);

    return rename(full_path, new_full_path) == 0 ? 1 : -1;
}

int Bridge_Delete(char *full_path) {
    int type = existance_checker(full_path);
    if (type == 0) { // Directory
        return remove_dir_recursive(full_path) == 0 ? 1 : -1;
    } else if (type == 1) { // File
        return remove(full_path) == 0 ? 1 : -1;
    }
    return 0; // Not found
}

int Bridge_CopyStruct(char *src_struct_path, char *dest_path) {
    // src_struct_path: .../structures/s_1
    // dest_path: /home/user/new_project
    // We want to copy contents of s_1 INTO new_project
    copy_dir_recursive(dest_path, src_struct_path);
    return 1;
}