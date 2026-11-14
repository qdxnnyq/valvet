# Velvet

A pink Valorant companion that lives on your PC. It reads the Riot Client already open there. It never asks for your password. A friend who installs it sees their own account, their own store, their own friends.

Windows may warn the first time. That is SmartScreen on an unsigned build, not a virus hit. The file is the release on this repo.

## Install

Download [VelvetSetup.exe](https://github.com/qdxnnyq/velvet/releases/latest/download/VelvetSetup.exe) and open it. It puts Velvet in the Start menu and on the desktop, then opens it. The Riot Client should be open on the same PC.

The window title follows Windows. If the PC is in dark mode, the frame around Velvet is dark too.

The next time there is a build, a pink bar shows up inside the app. One click downloads the new app and restarts. You do not have to run the installer again.

Settings stay in `%LOCALAPPDATA%\Velvet`. They are not baked into the file you send a friend. Backup opens a Gmail draft. Velvet never asks for the Gmail password. A new install starts at 15% volume.

## After the match

Win and loss sounds play when the **whole match** is over, from the final result. Not when a round ends. A round only flashes the edge of the window. Changing tabs does not play a sound.

GG, easy and Sad ship with the app. Cheer, fanfare, and a few quieter losses are in Settings too. You can drop in your own files. Preview plays them without waiting for a game.

## Sounds that are not the match

They each have their own setting.

- **Alert.** A friend joins the lobby, or the queue pops.
- **Friend in your game.** The sound when a friend is in your match. It can match the alert, or be its own pack or file.
- **Watch list in your game.** When someone on the watch list walks into your lobby or your game. The row glows. The overlay writes their name.
- **Watch list finishes a game.** When someone you watch, who does not have to be a friend, ends a match you were not in.
- **Wishlist.** A bell when a skin you pinned shows up in the daily store or the night market.

Quiet hours mute those. Win and loss still play after the match. Preview ignores quiet hours.

## What you actually open it for

**Live.** Names, agents, ranks, parties, and the server while you load in and while you play. The waiting animation stays under the lobby until a game is found. Today's challenge is one small line.

**History.** Best KDA on top, and it stays there while names load. Hover a player on the top of a finished scoreboard and the same row lights up below. MVP is the best player on the winning side. SVP is the best on the losing side. Open someone and their games show MVP too. Copy card turns the result into a picture.

**Profile.** Saved games on this PC, not the last handful. Win rate with Riot friends only, after two games together. Random teammates are not counted. Best kills, best ACS, the enemy you have met the most, your weapons, and your maps.

**Trace.** Not match history. Watch list, people who unfriended you, friend name changes (old name, new name), and a timed log of VP, Kingdom Credits, and Radianite.

**Find Duo.** Pick roles, modes, rank, and server first. Then press **Start looking for duo**. The page says you are looking. Stop looking takes you off. The id you share with one friend is random and stays on this PC. Discord is optional and never required.

**Store, bundle, esports, collection, agents, battle pass, patches, crosshairs.** The locker. Esports bundles open so you can see what is inside. Events slide across the top. Crosshairs are import and export, not read out of the game config. A background picture recolors the whole app from the photo.

**Overlay.** Optional, small, always on top. Score, and the watch-list names if any of them are in the lobby or the game.

**Live only.** A switch that loads nothing except the live match.

Velvet is not made by Riot Games. It does not inject into Valorant. It talks to the local Riot Client the same way the client talks to itself.
