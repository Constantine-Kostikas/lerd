import unittest

import historic_recipe

OLD = '''FROM php:{{.Version}}-fpm-alpine AS builder
RUN apk update && apk add --no-cache \\
        autoconf
# ── Runtime stage ──────
FROM php:{{.Version}}-fpm-alpine
RUN apk update && apk add --no-cache \\
        nodejs
{{.MkcertCA}}
'''


class Render(unittest.TestCase):
    def test_an_old_recipe_upgrades_in_the_runtime_stage_only(self):
        got = historic_recipe.render(OLD, '8.5')
        changed = [(a, b) for a, b in zip(OLD.replace('{{.Version}}', '8.5').replace('{{.MkcertCA}}', '').splitlines(), got.splitlines()) if a != b]
        self.assertEqual(changed, [('RUN apk update && apk add --no-cache \\', 'RUN apk update && apk upgrade --no-cache && apk add --no-cache \\')])
        builder, runtime = got.split('# ── Runtime stage')
        self.assertNotIn('apk upgrade', builder)
        self.assertIn('apk upgrade', runtime)

    def test_a_recipe_that_already_upgrades_is_left_alone(self):
        recipe = OLD.replace('FROM php:{{.Version}}-fpm-alpine\nRUN apk update && apk add', 'FROM php:{{.Version}}-fpm-alpine\nRUN apk update && apk upgrade --no-cache && apk add')
        self.assertEqual(historic_recipe.render(recipe, '8.5').count('apk upgrade'), 1)

    def test_placeholders_are_filled(self):
        got = historic_recipe.render(OLD, '8.4')
        self.assertNotIn('{{', got)
        self.assertIn('FROM php:8.4-fpm-alpine', got)

    def test_a_recipe_without_a_runtime_stage_is_refused(self):
        with self.assertRaises(ValueError):
            historic_recipe.render('FROM php:8.5-fpm-alpine\nRUN apk update && apk add --no-cache x\n', '8.5')


if __name__ == '__main__':
    unittest.main()
