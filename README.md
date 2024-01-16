<div align="center">

<pre>
 __ __ _ _   _   _  __  __  _  __  _ ___ ___   _____ ___  __   ____  _____ ___  
|  V  | | | | | | |/__\|  \| |/  \| | _ \ __| |_   _| _ \/  \ / _/ |/ / __| _ \ 
| \_/ | | |_| |_| | \/ | | ' | /\ | | v / _|    | | | v / /\ | \_|   <| _|| v / 
|_| |_|_|___|___|_|\__/|_|\__|_||_|_|_|_\___|   |_| |_|_\_||_|\__/_|\_\___|_|_\ 
-------------------------------------------------------------------------------
Golang web app to display "#millionaireinthemaking" progress
</pre>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
</div>

[@thejosephmurray](https://www.youtube.com/@thejosephmurray) has been tracking his daily income and expenditures for almost a year now and I wanted to visualize that progress through a graphical dashboard and perform an analysis to see how long it'll take for them to become a millionaire using solely profit and loss statements!

### Usage

Visit millionairetracker@domain.com (NOT DEPLOYED YET) to see the final site.

There are only two active pages which is: / and /analysis.
* / - data charts
* /analysis - forecast to become a millionaire and variance analysis

### System Design
![sys-design](https://github.com/epchao/millionaire-tracker/assets/46041923/d291785e-49f9-4a99-9969-1142b4a4c98d)


### Development setup

1. Download docker at https://www.docker.com/products/docker-desktop/

2. Clone this repository \
```git clone https://github.com/epchao/millionaire-tracker.git```

3. cd into it \
```cd millionaire-tracker```

4. Setup .env with your **DB_USER**, **DB_PASSWORD**, **DB_NAME**, and **TESSDATA_PREFIX** \
```touch .env```

5. Ensure you have an ./out/ folder to save processed images \
```mkdir out```

6. Build the docker container image and artifacts \
```docker build --no-cache .```

7. Run the docker container (remove -d to focus output) \
```docker compose up -d```

8. Run the Initialize script under ```./scripts/script.go``` inside of ```./api/main.go``` to populate the database

9. Remove Initialize script and replace it with Update script

### Reflection
* The pre-processing before the optical character recognition stage took many trial and error sessions due to finding the correct parameters and determining which techniques work best to retrieve the expenses and revenue text. It was a little more complicated due to the white background and the differently colored text but solved within the ./scripts/scripts.go file.
* Ensuring that this project was compatible with Docker was also a challenge because I had to ensure that FFMPEG, OPENCV, and tesseract were installed into the docker container. OpenCV includes 5.5 GB worth of functions to use, so I needed to have a pre-loaded image with OpenCV and fix all of the minor errors that appeared.

### Disclaimer
The datapoints may be inaccurate when the image's text wasn't read properly. The analysis is 100% based on sole revenue and expenses posted and doesnt include any practicial financial information.
