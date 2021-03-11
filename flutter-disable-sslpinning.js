function hook_ssl_verify_result(address)
{
    Interceptor.attach(address, {
        onEnter: function(args) {
            console.log("Disabling SSL validation")
        },
        onLeave: function(retval)
        {
            retval.replace(0x1);
        }
    });
}
function disablePinning()
{
    var pattern = "ff 03 05 d1 fc 6f 0f a9 f8 5f 10 a9 f6 57 11 a9 f4 4f 12 a9 fd 7b 13 a9 fd c3 04 91 08 0a 80 52"
    Process.enumerateRangesSync('r-x').filter(function (m) 
    {
        if (m.file) return m.file.path.indexOf('Flutter') > -1;
        return false;
    }).forEach(function (r) 
    {
        Memory.scanSync(r.base, r.size, pattern).forEach(function (match) {
        console.log('[+] ssl_verify_result found at: ' + match.address.toString());
        hook_ssl_verify_result(match.address);
        });
    });
}
setTimeout(disablePinning, 1000) 
