# \<Struct/>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## What is struct ?

In your coding journey you might have built a lot of projects and if these projects are using well known technolgies as a base then those technologies provide you an organized file structure with all the required necessities in place but if you are making projects with technologies which does not provide you any organized file structure or you are in need to make your own custom file structure then **\<struct/>** helps you do it.

### Key Features
- **Create once, Use it System-wide**: Once you create a file structure in the application, the structure will be noted and you can call the application and get the required file stucture anywhere.
- **Access the stucture from Anywhere**: In the future updates you will be made able to upload your structures to cloud and get those structures from anywhere.
- **Secured Downloads**: In the future updates, when you are trying to download structures you may choose to add a "give id to download" feature to protect the structes to be downloaded by trusted users

## Technologies Used

| Category       | Technologies/Tools                  |
|----------------|-------------------------------------|
| **Functionality of Application**    | C Language |
| **Package Manager**   | Golang                      |


## Repository Structure

```
struct/
├── code_for_struct/        # Where the actual program functionality resides
│   └── structures/         # Where all you file structures will be placed
│       ├──s_1/             # A structure example 
│       └──s_2/             # A structure example
│   ├── test_area/          # Where you can experiment new things
│   └── struct.c            # The implementation of the core functionality of the application
├── LICENCE                 # MIT Licence for this repository
└── README.md               # Recursion 
```

## Quick Start

### Prerequisites
- GCC
- Git

### Installation
1. **Clone the Repository**:
   ```bash
   git clone https://github.com/Saitej2456/struct.git
   cd struct
   cd code_for_struct
   ```

2. **Set the Path for structures directory**:
   - Open the struct.c file using a text editor of your choice (here vim is being used)
     ```bash
      vim struct.c
     ``` 
   - Go to main function and give the absolute path of your stuctures directory using the char path[PATH_MAX] string
     ```
      char path[PATH_MAX] = "<give_your_respective_absolute_path>"  
     ```

3. **Compile the C program**:
   ```bash
   gcc ./struct.c -o ./struct
   ```


4. **Copy the executable file to your bin directory**:
   ```bash
   sudo cp ./struct /bin/
   ```

5. **Access the App**:
   - Use the application from anywhere in your system using
     ```bash
      struct
     ``` 


##  Usage

### Main Intention
- **Create a structure**: Using this will ask you the name of the structure you want to create and then you can peform respective operations on your structures.
- **Use a structure**: Using this will show you a list of existing structures and ask you to confirm to move on by typing 'q' and then you will be asked the name of the structure to use [The respective chosen structure will be created where you called the struct command].
- **Remove a structure**: Using this will show you a list of existing structures and ask you to confirm to move on by typing 'q' and then you will be asked the name of the structure to be remove.
- **Edit a structure**: Using this will show you a list of existing structures and ask you to confirm to move on by typing 'q' and then you will be asked the name of the structure to be edit, then you will be able to perform respective operation on your structures.
- **Stop the program**: Using this will stop the application.

### Operations
- **Create a file**: A file will be created with the respective given name at mentioned current path.
- **Create a directory**: A directory will be created with the respective given name at mentioned current path.
- **Create a sript file**: A scipt file will be created with the respective given name at mentioned current path.
- **Move into a directory**: Mentioned current path will be updated by moving into a directory.
- **Move to parent directory**: Mentioned current path will be updated by moving into parent directory.
- **Remove a file**: A file will be removed with the respective given name at mentioned current path.
- **Remove a directory**: A directory will be removed with the respective given name at mentioned current path.
- **Rename a file**: A file will be renamed with the respective given name at mentioned current path.
- **Rename a directory**: A directory will be renamed with the respective given name at mentioned current path.
- **List current path contents**: All the contents will be listed at mentioned current path.
- **End making a structure**: Goes back to Main Intention phase.

## Testing

### Feature Testing
- Mention any feature that needs to be tested and how to test it here.

### Edge Cases Found
- Mention any edge cases found that needs to be addressed in here.

## Present and Future Updates

* [x] **Make a base functional application** 
* [ ] **Add upload stuctures feature**: Feature which allows user to choose an existing directory which will be stored as a structure.
* [ ] **Convert the way of stucture storage**: Currently whenever there is a structure being stored the entire file stucture is physically stored in "structures" directory.
* [ ] **Make a package manager**: A Package Manager which allows users to check stuctures available online and let them download or upload their own structure to the cloud.
* [ ] **Secure the structure download's access**: The structures need to be accessed only by the people for whom it is intended to be accessed.
* [ ] **Add a manuel page for the command:** Just like how linux commands give instructions on how to use them when a "--help" or "-h" parameter is used


## How to Contribute

1. Fork the repo.
2. Create a feature branch (`git checkout -b feature`).
3. Do respective changes in the **test_area** directory, only when features added to the test_area are confirmed to be safe to use, they will be added to the **struct.c** in the parent directory of **test_area** directory.
4. Commit changes (`git commit -m 'Added a feature X'`).
5. Push to branch (`git push origin feature`).
6. Open a Pull Request.

### Guidelines
- When creating **varaibles name**, do not use CAPS and if the variable has multiple words use '_' between the words.
- When creating **function names**, same rules as variable names apply.
- When creating **macro names** use all CAPS with '_' to distinguish words.
- Whether you are making new macros,variables,functions or defining new headers make sure they are all **declared in respective areas** which is written the source code using comments and make sure they all have **comments which tell about their intended usage**. 
- Add tests for new features.
- Update README if there are any changes.

##  License

This project is licensed under the MIT License.

## Contact

- **Mail**: [saitej2456@gmail.com](mailto:saitej2456@gmail.com)
- **GitHub**: [Saitej2456 [ofc its obv]](https://github.com/Saitej2456)
- **Issues**: Report bugs [here](https://github.com/Saitej2456/struct/issues).

---

⭐ **Star this repo!** ( because.... why not~ ? )