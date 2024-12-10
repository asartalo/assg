cd $ASSG_ROOT
echo "HELLO FROM pre.sh"
echo "ASSG_ROOT: $ASSG_ROOT"
pwd
ls -l 
if [ -f ./content/pre.txt ]; then
    rm ./content/pre.txt
fi
if [ -f ./content/post.txt ]; then
    rm ./content/post.txt
fi
foo=`ls ./content`
echo $foo
echo $foo > ./content/pre.txt
