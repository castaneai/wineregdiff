# wineregdiff

compare wine registry files and generate commands to apply the changes.

## Usage

```
$ cp $WINEPREFIX/user.reg user_old.reg
$ winetricks renderer=gdi
$ wineregdiff -root HKCU -force user_old.reg $WINEPREFIX/user.reg
wine REG ADD "HKEY_CURRENT_USER\\Software\\Wine\\Direct3D" /v "renderer" /t REG_SZ /d "gdi" /f
```
