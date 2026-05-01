# FL-AUD-01 Auditar ejecución local sin secretos

## Objetivo

Dejar evidencia operativa local de comandos, latencias, cola y errores para diagnosticar cuellos de botella entre proyectos.

## Secuencia

1. El CLI marca `startedAtUtc`.
2. El comando ejecuta o falla con error tipado.
3. El CLI registra un evento JSONL diario.
4. El operador puede ejecutar `audit export` o `audit summary`.

## Privacidad

La auditoría no guarda cuerpos de mensajes, captions, códigos, passwords, API hash, session blobs ni paths de archivos enviados.
