if [ -f ./content/pre.txt ]; then
    rm ./content/pre.txt
fi
if [ -f ./content/post.txt ]; then
    rm ./content/post.txt
fi
foo=`ls ./content`
echo $foo > ./content/pre.txt
