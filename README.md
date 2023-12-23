<div align="center">

<pre>
 __ __ _ _   _   _  __  __  _  __  _ ___ ___   _____ ___  __   ____  _____ ___  
|  V  | | | | | | |/__\|  \| |/  \| | _ \ __| |_   _| _ \/  \ / _/ |/ / __| _ \ 
| \_/ | | |_| |_| | \/ | | ' | /\ | | v / _|    | | | v / /\ | \_|   <| _|| v / 
|_| |_|_|___|___|_|\__/|_|\__|_||_|_|_|_\___|   |_| |_|_\_||_|\__/_|\_\___|_|_\ 
-------------------------------------------------------------------------------
{SHORT DESCRIPTION}
</pre>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
</div>

@todo: DESCRIPTION

## Installation

@todo INSTALL

## Usage example

@todo HOW TO USE

### Disclaimer

@todo CRUD OPS/ENDPOINTS

## Development setup

# MongoDB Atlas Golang Sample Project

This repository contains an example application that connects to MongoDB
Atlas using the Go MongoDB driver. You can use this application as
a reference for when you build your Golang application.

## Prerequisites

To build and run this project, you need Golang version 1.13.x or later.

## Getting Started

The following instructions explain how to get your project connected to your
instance of MongoDB Atlas.

### 1. Download the Repository

To get started with this sample project, download this repository to your
programming environment. You can clone this project using Git version control:

```
git clone https://github.com/mongodb-university/atlas_starter_go.git
```

Or you can download the ZIP archive using your browser. If you download
this project as a ZIP archive,
[unzip the archive](https://www.wikihow.com/Unzip-a-File) before proceeding.

### 2. Install Dependencies

Navigate to the directory containing the project in your shell and
install the necessary packages by running the following command.

```shell
go get -d
```

### 3. Configure your Atlas Credentials

1. Open the `main.go` file.

2. Search for the variable `mongoUri` near the top which is assigned
   placeholder text. Replace the placeholder text with the connection
   string for your Atlas cluster. For more information on finding the
   connection string, see [Connect via
   Driver](https://docs.atlas.mongodb.com/driver-connection/) in the Atlas
   documentation.

```go
	var mongoUri = "<Your Atlas Connection String>"
```

### 4. Run the Project

You can run the application from the directory that contains it with the
following command:

```shell
go run main.go
``
Assuming you have the correct connection string, you have now connected
the Go app to your MongoDB Atlas datastore. Have fun modifying the code to experiment with the Go driver and MongoDB.

## Troubleshooting

Are you having trouble getting connected to your MongoDB Atlas instance? Double-check the following:

1. Have you replaced the `mongoUri` variable with a valid connection string provided by the Atlas UI?

2. Have you [added your current IP address to the access list](https://docs.atlas.mongodb.com/security-whitelist/) in the Atlas UI?
```

## Credits

Distributed under the MIT license. See `LICENSE` for more information.

Created by: [Eugene Chao](https://github.com/epchao/)

## Contributing

1. Fork it (<https://github.com/epchao/{REPO}/fork>)
2. Create your feature branch (`git checkout -b feature/fooBar`)
3. Commit your changes (`git commit -am 'Add some fooBar'`)
4. Push to the branch (`git push origin feature/fooBar`)
5. Create a new Pull Request