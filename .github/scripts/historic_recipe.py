"""Render an older release's PHP-FPM recipe for a republish of its base image."""
import sys

PLACEHOLDERS = ('{{.CustomExtensions}}', '{{.CustomExtensionsRuntime}}',
                '{{.CustomPackages}}', '{{.MkcertCA}}')
RUNTIME_STAGE = '# ── Runtime stage'
RUNTIME_INSTALL = 'RUN apk update && apk add --no-cache'


def add_apk_upgrade(recipe):
    # Recipes before #2333 never upgrade what upstream php ships, so a republish
    # would keep its vulnerable packages. Only the runtime stage's install
    # changes; a recipe that already upgrades is left as it is.
    if 'apk upgrade' in recipe:
        return recipe
    at = recipe.index(RUNTIME_INSTALL, recipe.index(RUNTIME_STAGE))
    return recipe[:at] + 'RUN apk update && apk upgrade --no-cache && apk add --no-cache' + recipe[at + len(RUNTIME_INSTALL):]


def render(recipe, version):
    recipe = recipe.replace('{{.Version}}', version)
    for ph in PLACEHOLDERS:
        recipe = recipe.replace(ph, '')
    return add_apk_upgrade(recipe)


if __name__ == '__main__':
    version, src, dst = sys.argv[1:4]
    with open(src) as f:
        out = render(f.read(), version)
    with open(dst, 'w') as f:
        f.write(out)
