# utils 

a random bunch of binaries written to help me organize my files.

## nef to jpeg

```
apt-get install ufraw gimp-ufraw 
for file in $files; do 
  img=$(echo $file | sed 's/.NEF/.jpg/g')  
  dir=$(dirname $file)

  if [[ -e $img ]]; then 
    rm $file
    echo "exists"
  else 
    ufraw-batch --out-type=jpeg --out-path=$dir $file
    if [[ -e $img ]]; then
      rm $file
      echo removed $file
    fi
  fi
done
```
