# Cómo cambiar la contraseña de PostgreSQL

## Método 1: Usando sudo y psql (Recomendado)

```bash
# 1. Conectarse como superusuario del sistema
sudo -u postgres psql

# 2. Una vez dentro de psql, cambiar la contraseña
ALTER USER postgres PASSWORD 'nueva_contraseña';

# 3. Salir
\q
```

## Método 2: Modificando pg_hba.conf temporalmente

```bash
# 1. Encontrar y editar pg_hba.conf
sudo nano /Library/PostgreSQL/16/data/pg_hba.conf

# 2. Cambiar temporalmente la línea:
# De: local   all   postgres   md5
# A:  local   all   postgres   trust

# 3. Reiniciar PostgreSQL
sudo /Library/PostgreSQL/16/bin/pg_ctl restart -D /Library/PostgreSQL/16/data

# 4. Conectarse sin contraseña
/Library/PostgreSQL/16/bin/psql -U postgres

# 5. Cambiar la contraseña
ALTER USER postgres PASSWORD 'postgres';

# 6. Salir
\q

# 7. Revertir pg_hba.conf a su estado original (md5)
sudo nano /Library/PostgreSQL/16/data/pg_hba.conf

# 8. Reiniciar PostgreSQL nuevamente
sudo /Library/PostgreSQL/16/bin/pg_ctl restart -D /Library/PostgreSQL/16/data
```

## Método 3: Usando el comando directo (si tienes acceso)

```bash
/Library/PostgreSQL/16/bin/psql -U postgres -c "ALTER USER postgres PASSWORD 'postgres';"
```

## Método 4: Para macOS con PostgreSQL instalado via Homebrew

```bash
# Si PostgreSQL fue instalado con Homebrew
brew services stop postgresql@16
psql postgres -c "ALTER USER postgres PASSWORD 'postgres';"
brew services start postgresql@16
```

## Notas importantes:
- Reemplaza 'nueva_contraseña' con la contraseña que desees
- Después de cambiar la contraseña, asegúrate de reiniciar el servicio
- La nueva contraseña tomará efecto inmediatamente