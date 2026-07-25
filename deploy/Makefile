deps:
	ansible-galaxy install -r requirements.yml

check:
	ansible-playbook -J --check --diff -i inventory.ini site.yml

deploy:
	ansible-playbook -J -i inventory.ini site.yml

.PHONY: deps check deploy
