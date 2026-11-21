; Windows x64 forwarding stubs: preserve register and stack arguments.
EXTERN ResolveVersionProc:PROC
.code
PROXY_STUB MACRO number
PUBLIC VersionProxy&number
VersionProxy&number PROC FRAME
 sub rsp, 0A8h
 .allocstack 0A8h
 .endprolog
 mov [rsp+20h], rcx
 mov [rsp+28h], rdx
 mov [rsp+30h], r8
 mov [rsp+38h], r9
 movdqu [rsp+40h], xmm0
 movdqu [rsp+50h], xmm1
 movdqu [rsp+60h], xmm2
 movdqu [rsp+70h], xmm3
 mov ecx, number
 call ResolveVersionProc
 mov rcx, [rsp+20h]
 mov rdx, [rsp+28h]
 mov r8, [rsp+30h]
 mov r9, [rsp+38h]
 movdqu xmm0, [rsp+40h]
 movdqu xmm1, [rsp+50h]
 movdqu xmm2, [rsp+60h]
 movdqu xmm3, [rsp+70h]
 add rsp, 0A8h
 jmp rax
VersionProxy&number ENDP
ENDM
PROXY_STUB 0
PROXY_STUB 1
PROXY_STUB 2
PROXY_STUB 3
PROXY_STUB 4
PROXY_STUB 5
PROXY_STUB 6
PROXY_STUB 7
PROXY_STUB 8
PROXY_STUB 9
PROXY_STUB 10
PROXY_STUB 11
PROXY_STUB 12
PROXY_STUB 13
PROXY_STUB 14
PROXY_STUB 15
PROXY_STUB 16
END
